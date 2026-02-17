package ast

// Lower performs an AST-to-AST transformation pass that replaces high-level
// concurrency constructs (SpawnExpr, ParallelExpr, TryExpr) with their lowered
// equivalents. The lowered nodes carry pre-processed information that simplifies
// code generation.
//
// The pass walks the entire AST recursively, creating new nodes for transformed
// constructs while passing through everything else unchanged. The original AST
// is not mutated.
func Lower(prog *Program) *Program {
	l := &lowerer{}
	stmts, changed := l.lowerStmts(prog.Statements)
	if !changed {
		return prog
	}
	return &Program{
		Statements: stmts,
		SourceFile: prog.SourceFile,
		RawSource:  prog.RawSource,
		Structs:    prog.Structs,
	}
}

type lowerer struct{}

func (l *lowerer) lowerStmts(stmts []Statement) ([]Statement, bool) {
	var out []Statement
	modified := false
	for i, s := range stmts {
		ns := l.lowerStmt(s)
		if ns != s {
			if !modified {
				out = make([]Statement, len(stmts))
				copy(out[:i], stmts[:i])
				modified = true
			}
		}
		if modified {
			out[i] = ns
		}
	}
	if !modified {
		return stmts, false
	}
	return out, true
}

func (l *lowerer) lowerExprs(exprs []Expr) ([]Expr, bool) {
	var out []Expr
	modified := false
	for i, e := range exprs {
		ne := l.lowerExpr(e)
		if ne != e {
			if !modified {
				out = make([]Expr, len(exprs))
				copy(out[:i], exprs[:i])
				modified = true
			}
		}
		if modified {
			out[i] = ne
		}
	}
	if !modified {
		return exprs, false
	}
	return out, true
}

func (l *lowerer) lowerStmt(s Statement) Statement {
	switch st := s.(type) {
	case *FuncDef:
		params, pc := l.lowerParams(st.Params)
		body, bc := l.lowerStmts(st.Body)
		if !pc && !bc {
			return s
		}
		cp := *st
		cp.Params = params
		cp.Body = body
		return &cp

	case *TestDef:
		body, changed := l.lowerStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *BenchDef:
		body, changed := l.lowerStmts(st.Body)
		if !changed {
			return s
		}
		cp := *st
		cp.Body = body
		return &cp

	case *IfStmt:
		cond := l.lowerExpr(st.Condition)
		body, bc := l.lowerStmts(st.Body)
		elsifs, ec := l.lowerElsifs(st.ElsifClauses)
		elseBody, ebc := l.lowerStmts(st.ElseBody)
		if cond == st.Condition && !bc && !ec && !ebc {
			return s
		}
		cp := *st
		cp.Condition = cond
		cp.Body = body
		cp.ElsifClauses = elsifs
		cp.ElseBody = elseBody
		return &cp

	case *WhileStmt:
		cond := l.lowerExpr(st.Condition)
		body, bc := l.lowerStmts(st.Body)
		if cond == st.Condition && !bc {
			return s
		}
		cp := *st
		cp.Condition = cond
		cp.Body = body
		return &cp

	case *ForStmt:
		coll := l.lowerExpr(st.Collection)
		body, bc := l.lowerStmts(st.Body)
		if coll == st.Collection && !bc {
			return s
		}
		cp := *st
		cp.Collection = coll
		cp.Body = body
		return &cp

	case *ReturnStmt:
		if st.Value == nil {
			return s
		}
		val := l.lowerExpr(st.Value)
		if val == st.Value {
			return s
		}
		return &ReturnStmt{BaseStmt: st.BaseStmt, Value: val}

	case *ExprStmt:
		expr := l.lowerExpr(st.Expression)
		if expr == st.Expression {
			return s
		}
		return &ExprStmt{BaseStmt: st.BaseStmt, Expression: expr}

	case *AssignStmt:
		val := l.lowerExpr(st.Value)
		if val == st.Value {
			return s
		}
		cp := *st
		cp.Value = val
		return &cp

	case *IndexAssignStmt:
		obj := l.lowerExpr(st.Object)
		idx := l.lowerExpr(st.Index)
		val := l.lowerExpr(st.Value)
		if obj == st.Object && idx == st.Index && val == st.Value {
			return s
		}
		return &IndexAssignStmt{BaseStmt: st.BaseStmt, Object: obj, Index: idx, Value: val}

	case *DotAssignStmt:
		obj := l.lowerExpr(st.Object)
		val := l.lowerExpr(st.Value)
		if obj == st.Object && val == st.Value {
			return s
		}
		return &DotAssignStmt{BaseStmt: st.BaseStmt, Object: obj, Field: st.Field, Value: val}

	default:
		return s
	}
}

func (l *lowerer) lowerElsifs(clauses []ElsifClause) ([]ElsifClause, bool) {
	var out []ElsifClause
	modified := false
	for i, ec := range clauses {
		cond := l.lowerExpr(ec.Condition)
		body, bc := l.lowerStmts(ec.Body)
		if cond != ec.Condition || bc {
			if !modified {
				out = make([]ElsifClause, len(clauses))
				copy(out[:i], clauses[:i])
				modified = true
			}
		}
		if modified {
			out[i] = ElsifClause{Condition: cond, Body: body}
		}
	}
	if !modified {
		return clauses, false
	}
	return out, true
}

func (l *lowerer) lowerParams(params []Param) ([]Param, bool) {
	var out []Param
	modified := false
	for i, p := range params {
		if p.Default == nil {
			if modified {
				out[i] = p
			}
			continue
		}
		nd := l.lowerExpr(p.Default)
		if nd != p.Default {
			if !modified {
				out = make([]Param, len(params))
				copy(out[:i], params[:i])
				modified = true
			}
		}
		if modified {
			out[i] = Param{Name: p.Name, Default: nd}
		}
	}
	if !modified {
		return params, false
	}
	return out, true
}

func (l *lowerer) lowerExpr(e Expr) Expr {
	switch ex := e.(type) {
	// --- Lowering targets ---
	case *SpawnExpr:
		return l.lowerSpawn(ex)
	case *ParallelExpr:
		return l.lowerParallel(ex)
	case *TryExpr:
		return l.lowerTry(ex)

	// --- Recursive descent ---
	case *BinaryExpr:
		left := l.lowerExpr(ex.Left)
		right := l.lowerExpr(ex.Right)
		if left == ex.Left && right == ex.Right {
			return e
		}
		return &BinaryExpr{Left: left, Op: ex.Op, Right: right}

	case *UnaryExpr:
		operand := l.lowerExpr(ex.Operand)
		if operand == ex.Operand {
			return e
		}
		return &UnaryExpr{Op: ex.Op, Operand: operand}

	case *CallExpr:
		fn := l.lowerExpr(ex.Func)
		args, ac := l.lowerExprs(ex.Args)
		if fn == ex.Func && !ac {
			return e
		}
		return &CallExpr{Func: fn, Args: args}

	case *IndexExpr:
		obj := l.lowerExpr(ex.Object)
		idx := l.lowerExpr(ex.Index)
		if obj == ex.Object && idx == ex.Index {
			return e
		}
		return &IndexExpr{Object: obj, Index: idx}

	case *SliceExpr:
		obj := l.lowerExpr(ex.Object)
		start := l.lowerExpr(ex.Start)
		length := l.lowerExpr(ex.Length)
		if obj == ex.Object && start == ex.Start && length == ex.Length {
			return e
		}
		return &SliceExpr{Object: obj, Start: start, Length: length}

	case *DotExpr:
		obj := l.lowerExpr(ex.Object)
		if obj == ex.Object {
			return e
		}
		return &DotExpr{Object: obj, Field: ex.Field}

	case *ArrayLiteral:
		elems, changed := l.lowerExprs(ex.Elements)
		if !changed {
			return e
		}
		return &ArrayLiteral{Elements: elems}

	case *HashLiteral:
		pairs, changed := l.lowerPairs(ex.Pairs)
		if !changed {
			return e
		}
		return &HashLiteral{Pairs: pairs}

	case *FnExpr:
		params, pc := l.lowerParams(ex.Params)
		body, bc := l.lowerStmts(ex.Body)
		if !pc && !bc {
			return e
		}
		return &FnExpr{Params: params, Body: body}

	default:
		return e
	}
}

func (l *lowerer) lowerPairs(pairs []HashPair) ([]HashPair, bool) {
	var out []HashPair
	modified := false
	for i, p := range pairs {
		key := l.lowerExpr(p.Key)
		val := l.lowerExpr(p.Value)
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

// --- Lowering transforms ---

func (l *lowerer) lowerSpawn(e *SpawnExpr) Expr {
	body, _ := l.lowerStmts(e.Body)

	// Extract last ExprStmt as ResultExpr
	if len(body) > 0 {
		if es, ok := body[len(body)-1].(*ExprStmt); ok {
			return &LoweredSpawnExpr{
				Body:       body[:len(body)-1],
				ResultExpr: es.Expression,
			}
		}
	}
	return &LoweredSpawnExpr{Body: body}
}

func (l *lowerer) lowerParallel(e *ParallelExpr) Expr {
	branches := make([]ParallelBranch, len(e.Body))
	for i, s := range e.Body {
		ns := l.lowerStmt(s)
		if es, ok := ns.(*ExprStmt); ok {
			branches[i] = ParallelBranch{Expr: es.Expression, Index: i}
		} else {
			branches[i] = ParallelBranch{Stmts: []Statement{ns}, Index: i}
		}
	}
	return &LoweredParallelExpr{Branches: branches}
}

func (l *lowerer) lowerTry(e *TryExpr) Expr {
	expr := l.lowerExpr(e.Expr)
	handler, _ := l.lowerStmts(e.Handler)

	// Extract last ExprStmt as ResultExpr (simple case)
	if len(handler) > 0 {
		if es, ok := handler[len(handler)-1].(*ExprStmt); ok {
			return &LoweredTryExpr{
				Expr:       expr,
				ErrVar:     e.ErrVar,
				Handler:    handler[:len(handler)-1],
				ResultExpr: es.Expression,
			}
		}
	}
	// Complex case (IfStmt result) or empty handler: codegen handles it
	return &LoweredTryExpr{
		Expr:    expr,
		ErrVar:  e.ErrVar,
		Handler: handler,
	}
}
