package ast

// ImplicitReturnLowering returns a Transform that converts last-expression-as-return-value
// patterns into explicit AST nodes. This moves implicit return detection from codegen
// into the transform pipeline.
//
// The transform handles three contexts:
//   - FuncDef.Body: last ExprStmt → ImplicitReturnStmt
//   - FnExpr.Body: last ExprStmt → ImplicitReturnStmt
//   - LoweredTryExpr.Handler (when ResultExpr is nil): last ExprStmt → TryResultStmt
//
// When the last statement is an IfStmt, the transform recurses into each branch,
// converting the last ExprStmt in each branch to the appropriate node type.
func ImplicitReturnLowering() Transform {
	return TransformFunc{
		N: "implicit-return-lowering",
		F: func(prog *Program) *Program {
			ir := &implicitReturner{f: NewFactory()}
			stmts, changed := ir.walkStmts(prog.Statements)
			if !changed {
				return prog
			}
			return ir.f.ProgramFrom(prog, stmts)
		},
	}
}

type implicitReturner struct {
	f *Factory
}

// walkStmts walks a slice of statements, looking for FuncDef and other
// containers that need implicit return processing.
func (ir *implicitReturner) walkStmts(stmts []Statement) ([]Statement, bool) {
	return mapSlice(stmts, ir.walkStmt)
}

// walkStmt processes a single statement, recursing into containers.
func (ir *implicitReturner) walkStmt(s Statement) Statement {
	switch st := s.(type) {
	case *FuncDef:
		body := ir.transformBody(st.Body, implicitReturnContext)
		if body == nil {
			return s
		}
		return ir.f.FuncDefWithBody(st, body)

	case *TestDef:
		body, changed := ir.walkStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *BenchDef:
		body, changed := ir.walkStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *IfStmt:
		body, bc := ir.walkStmts(st.Body)
		elsifs, ec := ir.walkElsifs(st.ElsifClauses)
		elseBody, ebc := ir.walkStmts(st.ElseBody)
		if !bc && !ec && !ebc {
			return s
		}
		return ir.f.IfStmtWithBranches(st, body, elseBody, elsifs)

	case *CaseStmt:
		ofs, oc := ir.walkOfClauses(st.OfClauses)
		elsifs, ec := ir.walkElsifs(st.ElsifClauses)
		elseBody, ebc := ir.walkStmts(st.ElseBody)
		if !oc && !ec && !ebc {
			return s
		}
		return ir.f.CaseStmtWithBranches(st, ofs, elsifs, elseBody)

	case *WhileStmt:
		body, changed := ir.walkStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *ForStmt:
		body, changed := ir.walkStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *ExprStmt:
		expr := ir.walkExpr(st.Expression)
		if expr == st.Expression {
			return s
		}
		return &ExprStmt{BaseStmt: st.BaseStmt, Expression: expr}

	case *AssignStmt:
		val := ir.walkExpr(st.Value)
		if val == st.Value {
			return s
		}
		cp := *st
		cp.Value = val
		return &cp

	case *ReturnStmt:
		if st.Value == nil {
			return s
		}
		val := ir.walkExpr(st.Value)
		if val == st.Value {
			return s
		}
		return &ReturnStmt{BaseStmt: st.BaseStmt, Value: val}

	case *IndexAssignStmt:
		obj := ir.walkExpr(st.Object)
		idx := ir.walkExpr(st.Index)
		val := ir.walkExpr(st.Value)
		if obj == st.Object && idx == st.Index && val == st.Value {
			return s
		}
		return &IndexAssignStmt{BaseStmt: st.BaseStmt, Object: obj, Index: idx, Value: val}

	case *DotAssignStmt:
		obj := ir.walkExpr(st.Object)
		val := ir.walkExpr(st.Value)
		if obj == st.Object && val == st.Value {
			return s
		}
		return &DotAssignStmt{BaseStmt: st.BaseStmt, Object: obj, Field: st.Field, Value: val}

	default:
		return s
	}
}

// walkElsifs processes elsif clauses, recursing into their bodies.
func (ir *implicitReturner) walkElsifs(clauses []ElsifClause) ([]ElsifClause, bool) {
	var out []ElsifClause
	modified := false
	for i, ec := range clauses {
		body, changed := ir.walkStmts(ec.Body)
		if changed {
			if !modified {
				out = make([]ElsifClause, len(clauses))
				copy(out[:i], clauses[:i])
				modified = true
			}
		}
		if modified {
			out[i] = ElsifClause{Condition: ec.Condition, Body: body}
		}
	}
	if !modified {
		return clauses, false
	}
	return out, true
}

// walkExpr recurses into expressions that may contain bodies needing
// implicit return processing (FnExpr, LoweredTryExpr, LoweredSpawnExpr).
func (ir *implicitReturner) walkExpr(e Expr) Expr {
	switch ex := e.(type) {
	case *FnExpr:
		body := ir.transformBody(ex.Body, implicitReturnContext)
		if body == nil {
			return e
		}
		return ir.f.FnExprWithBody(ex, body)

	case *LoweredTryExpr:
		// Walk the tried expression
		expr := ir.walkExpr(ex.Expr)

		if ex.ResultExpr == nil && len(ex.Handler) > 0 {
			// No pre-extracted result — apply try-result lowering to handler body
			handler := ir.transformBody(ex.Handler, tryResultContext)
			if handler != nil || expr != ex.Expr {
				h := ex.Handler
				if handler != nil {
					h = handler
				}
				return ir.f.LoweredTry(expr, ex.ErrVar, h, nil)
			}
			return e
		}
		// ResultExpr already set or empty handler — just walk sub-expressions
		if expr != ex.Expr {
			return ir.f.LoweredTry(expr, ex.ErrVar, ex.Handler, ex.ResultExpr)
		}
		return e

	case *LoweredSpawnExpr:
		// Walk body statements for nested expressions
		body, changed := ir.walkStmts(ex.Body)
		if ex.ResultExpr != nil {
			resultExpr := ir.walkExpr(ex.ResultExpr)
			if changed || resultExpr != ex.ResultExpr {
				return ir.f.LoweredSpawn(body, resultExpr)
			}
			return e
		}
		if changed {
			return ir.f.LoweredSpawn(body, nil)
		}
		return e

	case *LoweredParallelExpr:
		branches, changed := ir.walkBranches(ex.Branches)
		if !changed {
			return e
		}
		return ir.f.LoweredParallel(branches)

	case *BinaryExpr:
		left := ir.walkExpr(ex.Left)
		right := ir.walkExpr(ex.Right)
		if left == ex.Left && right == ex.Right {
			return e
		}
		return &BinaryExpr{Left: left, Op: ex.Op, Right: right}

	case *UnaryExpr:
		operand := ir.walkExpr(ex.Operand)
		if operand == ex.Operand {
			return e
		}
		return &UnaryExpr{Op: ex.Op, Operand: operand}

	case *CallExpr:
		fn := ir.walkExpr(ex.Func)
		args, ac := ir.walkExprs(ex.Args)
		if fn == ex.Func && !ac {
			return e
		}
		return &CallExpr{Func: fn, Args: args}

	case *IndexExpr:
		obj := ir.walkExpr(ex.Object)
		idx := ir.walkExpr(ex.Index)
		if obj == ex.Object && idx == ex.Index {
			return e
		}
		return &IndexExpr{Object: obj, Index: idx}

	case *SliceExpr:
		obj := ir.walkExpr(ex.Object)
		start := ir.walkExpr(ex.Start)
		length := ir.walkExpr(ex.Length)
		if obj == ex.Object && start == ex.Start && length == ex.Length {
			return e
		}
		return &SliceExpr{Object: obj, Start: start, Length: length}

	case *DotExpr:
		obj := ir.walkExpr(ex.Object)
		if obj == ex.Object {
			return e
		}
		return &DotExpr{Object: obj, Field: ex.Field}

	case *ArrayLiteral:
		elems, changed := ir.walkExprs(ex.Elements)
		if !changed {
			return e
		}
		return &ArrayLiteral{Elements: elems}

	case *HashLiteral:
		pairs, changed := ir.walkPairs(ex.Pairs)
		if !changed {
			return e
		}
		return &HashLiteral{Pairs: pairs}

	case *CaseExpr:
		subject := ir.walkExpr(ex.Subject)
		ofClauses, ofChanged := ir.walkOfClauses(ex.OfClauses)
		elsifs, elsifChanged := ir.walkElsifs(ex.ElsifClauses)
		elseBody, elseChanged := ir.walkStmts(ex.ElseBody)
		if subject == ex.Subject && !ofChanged && !elsifChanged && !elseChanged {
			return e
		}
		return ir.f.CaseExprWithBranches(
			&CaseExpr{Subject: subject, SourceLine: ex.SourceLine},
			ofClauses, elsifs, elseBody,
		)

	default:
		return e
	}
}

func (ir *implicitReturner) walkExprs(exprs []Expr) ([]Expr, bool) {
	return mapSlice(exprs, ir.walkExpr)
}

func (ir *implicitReturner) walkPairs(pairs []HashPair) ([]HashPair, bool) {
	var out []HashPair
	modified := false
	for i, p := range pairs {
		key := ir.walkExpr(p.Key)
		val := ir.walkExpr(p.Value)
		if key != p.Key || val != p.Value {
			if !modified {
				out = make([]HashPair, len(pairs))
				copy(out[:i], pairs[:i])
				modified = true
			}
		}
		if modified {
			out[i] = HashPair{Key: key, Value: val}
		}
	}
	if !modified {
		return pairs, false
	}
	return out, true
}

func (ir *implicitReturner) walkBranches(branches []ParallelBranch) ([]ParallelBranch, bool) {
	var out []ParallelBranch
	modified := false
	for i, br := range branches {
		if br.Expr != nil {
			ne := ir.walkExpr(br.Expr)
			if ne != br.Expr {
				if !modified {
					out = make([]ParallelBranch, len(branches))
					copy(out[:i], branches[:i])
					modified = true
				}
			}
			if modified {
				out[i] = ParallelBranch{Expr: ne, Index: br.Index}
			}
		} else {
			stmts, sc := ir.walkStmts(br.Stmts)
			if sc {
				if !modified {
					out = make([]ParallelBranch, len(branches))
					copy(out[:i], branches[:i])
					modified = true
				}
			}
			if modified {
				out[i] = ParallelBranch{Stmts: stmts, Index: br.Index}
			}
		}
	}
	if !modified {
		return branches, false
	}
	return out, true
}

// bodyContext determines what node type to create for implicit returns.
type bodyContext int

const (
	implicitReturnContext bodyContext = iota // function/lambda → ImplicitReturnStmt
	tryResultContext                         // try handler → TryResultStmt
)

// transformBody converts the last expression in a body to the appropriate
// implicit return node. Returns nil if no transformation was needed.
func (ir *implicitReturner) transformBody(body []Statement, ctx bodyContext) []Statement {
	if len(body) == 0 {
		return nil
	}

	// First, walk all statements to process nested containers
	walked, changed := ir.walkStmts(body)

	lastIdx := len(walked) - 1
	last := walked[lastIdx]

	replaced := ir.transformLastStmt(last, ctx)
	if replaced == nil && !changed {
		return nil
	}

	// Need to create new slice if not already created by walkStmts
	if !changed {
		out := make([]Statement, len(walked))
		copy(out, walked)
		walked = out
	}
	if replaced != nil {
		walked[lastIdx] = replaced
	}
	return walked
}

// transformLastStmt converts a statement to an implicit return node.
// For ExprStmt, creates the appropriate return/result node.
// For IfStmt, recurses into each branch.
// Returns nil if no transformation was made.
func (ir *implicitReturner) transformLastStmt(s Statement, ctx bodyContext) Statement {
	switch st := s.(type) {
	case *ExprStmt:
		return ir.makeReturnNode(st.Expression, st.StmtLine(), ctx)

	case *IfStmt:
		return ir.transformIfBranches(st, ctx)

	case *CaseStmt:
		return ir.transformCaseBranches(st, ctx)

	default:
		return nil
	}
}

// makeReturnNode creates the appropriate implicit return node for the given context.
func (ir *implicitReturner) makeReturnNode(value Expr, line int, ctx bodyContext) Statement {
	switch ctx {
	case tryResultContext:
		return ir.f.TryResult(value, line)
	default:
		return ir.f.ImplicitReturn(value, line)
	}
}

// transformIfBranches recurses into an IfStmt's branches, converting the
// last expression in each branch to the appropriate return node.
func (ir *implicitReturner) transformIfBranches(ifStmt *IfStmt, ctx bodyContext) Statement {
	body := ir.transformBranchBody(ifStmt.Body, ctx)
	elsifs := ir.transformElsifBranches(ifStmt.ElsifClauses, ctx)
	elseBody := ir.transformBranchBody(ifStmt.ElseBody, ctx)

	if body == nil && elsifs == nil && elseBody == nil {
		return nil
	}

	b := ifStmt.Body
	if body != nil {
		b = body
	}
	e := ifStmt.ElseBody
	if elseBody != nil {
		e = elseBody
	}
	ec := ifStmt.ElsifClauses
	if elsifs != nil {
		ec = elsifs
	}
	return ir.f.IfStmtWithBranches(ifStmt, b, e, ec)
}

// transformBranchBody converts the last expression in a branch body to
// the appropriate return node. Returns nil if no transformation was needed.
func (ir *implicitReturner) transformBranchBody(body []Statement, ctx bodyContext) []Statement {
	if len(body) == 0 {
		return nil
	}
	lastIdx := len(body) - 1
	replaced := ir.transformLastStmt(body[lastIdx], ctx)
	if replaced == nil {
		return nil
	}
	out := make([]Statement, len(body))
	copy(out, body)
	out[lastIdx] = replaced
	return out
}

// transformElsifBranches converts the last expression in each elsif clause.
func (ir *implicitReturner) transformElsifBranches(clauses []ElsifClause, ctx bodyContext) []ElsifClause {
	var out []ElsifClause
	modified := false
	for i, ec := range clauses {
		body := ir.transformBranchBody(ec.Body, ctx)
		if body != nil {
			if !modified {
				out = make([]ElsifClause, len(clauses))
				copy(out[:i], clauses[:i])
				modified = true
			}
		}
		if modified {
			b := ec.Body
			if body != nil {
				b = body
			}
			out[i] = ElsifClause{Condition: ec.Condition, Body: b}
		}
	}
	if !modified {
		return nil
	}
	return out
}

// walkOfClauses processes of clauses, recursing into their bodies.
func (ir *implicitReturner) walkOfClauses(clauses []OfClause) ([]OfClause, bool) {
	var out []OfClause
	modified := false
	for i, oc := range clauses {
		if oc.ArrowExpr != nil {
			// Arrow form: walk the expression
			expr := ir.walkExpr(oc.ArrowExpr)
			if expr != oc.ArrowExpr {
				if !modified {
					out = make([]OfClause, len(clauses))
					copy(out[:i], clauses[:i])
					modified = true
				}
			}
			if modified {
				out[i] = OfClause{Values: oc.Values, ArrowExpr: expr}
			}
		} else {
			// Multi-line body form
			body, changed := ir.walkStmts(oc.Body)
			if changed {
				if !modified {
					out = make([]OfClause, len(clauses))
					copy(out[:i], clauses[:i])
					modified = true
				}
			}
			if modified {
				out[i] = OfClause{Values: oc.Values, Body: body}
			}
		}
	}
	if !modified {
		return clauses, false
	}
	return out, true
}

// transformCaseBranches recurses into a CaseStmt's branches, converting the
// last expression in each branch to the appropriate return node.
func (ir *implicitReturner) transformCaseBranches(cs *CaseStmt, ctx bodyContext) Statement {
	ofs := ir.transformOfBranches(cs.OfClauses, ctx)
	elsifs := ir.transformElsifBranches(cs.ElsifClauses, ctx)
	elseBody := ir.transformBranchBody(cs.ElseBody, ctx)

	if ofs == nil && elsifs == nil && elseBody == nil {
		return nil
	}

	o := cs.OfClauses
	if ofs != nil {
		o = ofs
	}
	ec := cs.ElsifClauses
	if elsifs != nil {
		ec = elsifs
	}
	e := cs.ElseBody
	if elseBody != nil {
		e = elseBody
	}
	return ir.f.CaseStmtWithBranches(cs, o, ec, e)
}

// transformOfBranches converts the last expression in each of clause.
func (ir *implicitReturner) transformOfBranches(clauses []OfClause, ctx bodyContext) []OfClause {
	var out []OfClause
	modified := false
	for i, oc := range clauses {
		if oc.ArrowExpr != nil {
			// Arrow form: wrap the expression as a return node
			ret := ir.makeReturnNode(oc.ArrowExpr, 0, ctx)
			if ret != nil {
				if !modified {
					out = make([]OfClause, len(clauses))
					copy(out[:i], clauses[:i])
					modified = true
				}
			}
			if modified {
				if ret != nil {
					out[i] = OfClause{Values: oc.Values, Body: []Statement{ret}}
				} else {
					out[i] = oc
				}
			}
		} else {
			body := ir.transformBranchBody(oc.Body, ctx)
			if body != nil {
				if !modified {
					out = make([]OfClause, len(clauses))
					copy(out[:i], clauses[:i])
					modified = true
				}
			}
			if modified {
				b := oc.Body
				if body != nil {
					b = body
				}
				out[i] = OfClause{Values: oc.Values, Body: b}
			}
		}
	}
	if !modified {
		return nil
	}
	return out
}
