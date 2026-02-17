package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
)

// buildStmt converts an AST statement into Go output AST nodes.
// Returns a slice (line directive + statement). Side effects (scope
// tracking, constant checks) happen during building.
func (g *codeGen) buildStmt(s ast.Statement) ([]GoStmt, error) {
	// Temporarily switch source file for statements from required files.
	if src := s.StmtSource(); src != "" && src != g.sourceFile {
		saved := g.sourceFile
		g.sourceFile = src
		g.w.source = src
		defer func() { g.sourceFile = saved; g.w.source = saved }()
	}

	var stmts []GoStmt
	if line := s.StmtLine(); line > 0 && g.sourceFile != "" {
		stmts = append(stmts, GoLineDirective{File: g.sourceFile, Line: line})
	}

	node, err := g.buildStmtInner(s)
	if err != nil {
		return nil, g.stmtError(s, err)
	}
	if node != nil {
		stmts = append(stmts, node...)
	}
	return stmts, nil
}

// buildStmts builds a body of statements into Go output AST nodes.
func (g *codeGen) buildStmts(stmts []ast.Statement) ([]GoStmt, error) {
	var out []GoStmt
	for _, s := range stmts {
		nodes, err := g.buildStmt(s)
		if err != nil {
			return nil, err
		}
		out = append(out, nodes...)
	}
	return out, nil
}

func (g *codeGen) buildStmtInner(s ast.Statement) ([]GoStmt, error) {
	switch st := s.(type) {
	case *ast.AssignStmt:
		return g.buildAssign(st)
	case *ast.IndexAssignStmt:
		return g.buildIndexAssign(st)
	case *ast.DotAssignStmt:
		return g.buildDotAssign(st)
	case *ast.ExprStmt:
		return g.buildExprStmt(st)
	case *ast.IfStmt:
		return g.buildIf(st)
	case *ast.WhileStmt:
		return g.buildWhile(st)
	case *ast.ForStmt:
		return g.buildFor(st)
	case *ast.BreakStmt:
		return []GoStmt{GoBreakStmt{}}, nil
	case *ast.NextStmt:
		return []GoStmt{GoContinueStmt{}}, nil
	case *ast.ReturnStmt:
		return g.buildReturn(st)
	case *ast.ImplicitReturnStmt:
		return g.buildImplicitReturn(st)
	case *ast.TryResultStmt:
		return g.buildTryResult(st)
	case *ast.SpawnReturnStmt:
		return g.buildSpawnReturn(st)
	case *ast.TryHandlerReturnStmt:
		return g.buildTryHandlerReturn(st)
	case *ast.FuncDef:
		return nil, fmt.Errorf("nested function definitions not supported")
	case *ast.RequireStmt:
		return nil, nil
	case *ast.ImportStmt:
		return nil, nil
	case *ast.SandboxStmt:
		return nil, fmt.Errorf("sandbox must be a top-level directive — it cannot appear inside functions, blocks, or control flow")
	default:
		return nil, fmt.Errorf("unknown statement type: %T", s)
	}
}

// --- Leaf statement builders ---

func (g *codeGen) buildAssign(a *ast.AssignStmt) ([]GoStmt, error) {
	// Uppercase names are constants — reject reassignment
	if origLine, ok := g.constantLine(a.Target); ok {
		return nil, fmt.Errorf("cannot reassign constant %s (first assigned at line %d)", a.Target, origLine)
	}

	exprType := g.exprType(a.Value)
	varType := g.varType(a.Target)

	isCaptured := g.isCapturedVar(a.Target)
	if isCaptured {
		varType = g.capturedVarType(a.Target)
	}

	expr, err := g.exprString(a.Value)
	if err != nil {
		return nil, err
	}
	if !varType.IsTyped() && exprType.IsTyped() {
		expr = fmt.Sprintf("interface{}(%s)", expr)
	}
	if isCaptured && varType.IsTyped() && !exprType.IsTyped() {
		switch varType {
		case TypeInt:
			expr = fmt.Sprintf("rugo_to_int(%s)", expr)
		case TypeFloat:
			expr = fmt.Sprintf("rugo_to_float(%s)", expr)
		case TypeString:
			expr = fmt.Sprintf("rugo_to_string(%s)", expr)
		case TypeBool:
			expr = fmt.Sprintf("rugo_to_bool(%s)", expr)
		}
	}

	var stmts []GoStmt
	op := ":="
	if g.isDeclared(a.Target) || (g.handlerVars[a.Target] && !g.inFunc) {
		op = "="
		if g.handlerVars[a.Target] && !g.isDeclared(a.Target) {
			g.declareVar(a.Target)
			if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
				g.declareConst(a.Target, a.SourceLine)
			}
		}
	} else {
		g.declareVar(a.Target)
		if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
			g.declareConst(a.Target, a.SourceLine)
		}
	}
	stmts = append(stmts, GoAssignStmt{Target: a.Target, Op: op, Value: GoRawExpr{Code: expr}})
	stmts = append(stmts, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", a.Target)}})
	return stmts, nil
}

func (g *codeGen) buildIndexAssign(ia *ast.IndexAssignStmt) ([]GoStmt, error) {
	obj, err := g.exprString(ia.Object)
	if err != nil {
		return nil, err
	}
	idx, err := g.exprString(ia.Index)
	if err != nil {
		return nil, err
	}
	val, err := g.exprString(ia.Value)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("rugo_index_set(%s, %s, %s)", obj, idx, val)}}}, nil
}

func (g *codeGen) buildDotAssign(da *ast.DotAssignStmt) ([]GoStmt, error) {
	if da.Field == "__type__" {
		return nil, fmt.Errorf("cannot assign to .__type__ — use type_of() for type introspection")
	}
	obj, err := g.exprString(da.Object)
	if err != nil {
		return nil, err
	}
	val, err := g.exprString(da.Value)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("rugo_dot_set(%s, %q, %s)", obj, da.Field, val)}}}, nil
}

func (g *codeGen) buildExprStmt(e *ast.ExprStmt) ([]GoStmt, error) {
	expr, err := g.exprString(e.Expression)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", expr)}}}, nil
}

func (g *codeGen) buildReturn(r *ast.ReturnStmt) ([]GoStmt, error) {
	fti := g.currentFuncTypeInfo()
	if r.Value == nil {
		if fti != nil && fti.ReturnType.IsTyped() {
			return []GoStmt{GoReturnStmt{Value: GoRawExpr{Code: typedZero(fti.ReturnType)}}}, nil
		}
		return []GoStmt{GoReturnStmt{Value: GoRawExpr{Code: "nil"}}}, nil
	}
	expr, err := g.exprString(r.Value)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoReturnStmt{Value: GoRawExpr{Code: expr}}}, nil
}

func (g *codeGen) buildImplicitReturn(r *ast.ImplicitReturnStmt) ([]GoStmt, error) {
	expr, err := g.exprString(r.Value)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoReturnStmt{Value: GoRawExpr{Code: expr}}}, nil
}

func (g *codeGen) buildTryResult(r *ast.TryResultStmt) ([]GoStmt, error) {
	expr, err := g.exprString(r.Value)
	if err != nil {
		return nil, err
	}
	return []GoStmt{GoAssignStmt{Target: "r", Op: "=", Value: GoRawExpr{Code: expr}}}, nil
}

func (g *codeGen) buildSpawnReturn(r *ast.SpawnReturnStmt) ([]GoStmt, error) {
	var stmts []GoStmt
	if r.Value != nil {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, GoAssignStmt{Target: "t.result", Op: "=", Value: GoRawExpr{Code: expr}})
	}
	stmts = append(stmts, GoReturnStmt{})
	return stmts, nil
}

func (g *codeGen) buildTryHandlerReturn(r *ast.TryHandlerReturnStmt) ([]GoStmt, error) {
	var stmts []GoStmt
	if r.Value != nil {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, GoAssignStmt{Target: "r", Op: "=", Value: GoRawExpr{Code: expr}})
	}
	stmts = append(stmts, GoReturnStmt{})
	return stmts, nil
}

// --- Container statement builders ---

func (g *codeGen) buildIf(i *ast.IfStmt) ([]GoStmt, error) {
	// Pre-declare variables (Ruby-like scoping)
	var preDecls []GoStmt
	var allBranches []ast.Statement
	allBranches = append(allBranches, i.Body...)
	for _, ec := range i.ElsifClauses {
		allBranches = append(allBranches, ec.Body...)
	}
	allBranches = append(allBranches, i.ElseBody...)
	for _, name := range collectAssignTargets(allBranches) {
		if !g.isDeclared(name) {
			varType := g.varType(name)
			if varType.IsTyped() {
				preDecls = append(preDecls, GoRawStmt{Code: fmt.Sprintf("var %s %s", name, varType.GoType())})
			} else {
				preDecls = append(preDecls, GoRawStmt{Code: fmt.Sprintf("var %s interface{}", name)})
			}
			g.declareVar(name)
		}
	}

	cond, err := g.exprString(i.Condition)
	if err != nil {
		return nil, err
	}

	body, err := g.buildStmts(i.Body)
	if err != nil {
		return nil, err
	}

	var elseIfs []GoElseIf
	for _, ec := range i.ElsifClauses {
		ecCond, err := g.exprString(ec.Condition)
		if err != nil {
			return nil, err
		}
		ecBody, err := g.buildStmts(ec.Body)
		if err != nil {
			return nil, err
		}
		elseIfs = append(elseIfs, GoElseIf{
			Cond: GoRawExpr{Code: g.condExpr(ecCond, ec.Condition)},
			Body: ecBody,
		})
	}

	var elseBody []GoStmt
	if len(i.ElseBody) > 0 {
		elseBody, err = g.buildStmts(i.ElseBody)
		if err != nil {
			return nil, err
		}
	}

	ifStmt := GoIfStmt{
		Cond:   GoRawExpr{Code: g.condExpr(cond, i.Condition)},
		Body:   body,
		ElseIf: elseIfs,
		Else:   elseBody,
	}

	result := append(preDecls, ifStmt)
	return result, nil
}

func (g *codeGen) buildWhile(w *ast.WhileStmt) ([]GoStmt, error) {
	cond, err := g.exprString(w.Condition)
	if err != nil {
		return nil, err
	}

	g.pushScope()
	body, err := g.buildStmts(w.Body)
	if err != nil {
		return nil, err
	}
	g.popScope()

	return []GoStmt{GoForStmt{
		Cond: g.condExpr(cond, w.Condition),
		Body: body,
	}}, nil
}

func (g *codeGen) buildFor(f *ast.ForStmt) ([]GoStmt, error) {
	// Try optimized integer range loop
	startExpr, endExpr := g.rangeExprs(f.Collection)
	if startExpr != "" {
		return g.buildForRange(f, startExpr, endExpr)
	}

	coll, err := g.exprString(f.Collection)
	if err != nil {
		return nil, err
	}

	iterVar := f.Var
	idxVar := f.IndexVar

	g.pushScope()
	var preamble []GoStmt

	if idxVar != "" {
		// Two-variable form: for key, val in hash / for idx, val in arr
		if iterVar != "_" {
			preamble = append(preamble, GoAssignStmt{Target: iterVar, Op: ":=", Value: GoRawExpr{Code: "rugo_for_kv.Key"}})
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", iterVar)}})
			g.declareVar(iterVar)
		} else {
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: "_ = rugo_for_kv.Key"}})
		}
		if idxVar != "_" {
			preamble = append(preamble, GoAssignStmt{Target: idxVar, Op: ":=", Value: GoRawExpr{Code: "rugo_for_kv.Val"}})
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", idxVar)}})
			g.declareVar(idxVar)
		} else {
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: "_ = rugo_for_kv.Val"}})
		}

		body, err := g.buildStmts(f.Body)
		if err != nil {
			return nil, err
		}
		g.popScope()

		return []GoStmt{GoForRangeStmt{
			Key:        "_",
			Value:      "rugo_for_kv",
			Collection: GoRawExpr{Code: fmt.Sprintf("rugo_iterable(%s)", coll)},
			Body:       append(preamble, body...),
		}}, nil
	}

	// Single-variable form
	preamble = append(preamble, GoAssignStmt{Target: iterVar, Op: ":=", Value: GoRawExpr{Code: "rugo_for_item"}})
	preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", iterVar)}})
	g.declareVar(iterVar)

	body, err := g.buildStmts(f.Body)
	if err != nil {
		return nil, err
	}
	g.popScope()

	return []GoStmt{GoForRangeStmt{
		Key:        "_",
		Value:      "rugo_for_item",
		Collection: GoRawExpr{Code: fmt.Sprintf("rugo_iterable_default(%s)", coll)},
		Body:       append(preamble, body...),
	}}, nil
}

func (g *codeGen) buildForRange(f *ast.ForStmt, startExpr, endExpr string) ([]GoStmt, error) {
	iterVar := f.Var
	idxVar := f.IndexVar

	g.pushScope()
	var preamble []GoStmt

	var forStmt GoStmt
	if idxVar != "" {
		// Two-variable form: for idx, val in range(5, 20)
		if iterVar != "_" {
			preamble = append(preamble, GoAssignStmt{Target: iterVar, Op: ":=", Value: GoRawExpr{Code: "rugo_range_idx"}})
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", iterVar)}})
			g.declareVar(iterVar)
		} else {
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: "_ = rugo_range_idx"}})
		}
		if idxVar != "_" {
			preamble = append(preamble, GoAssignStmt{Target: idxVar, Op: ":=", Value: GoRawExpr{Code: "rugo_range_i"}})
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", idxVar)}})
			g.declareVar(idxVar)
		} else {
			preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: "_ = rugo_range_i"}})
		}

		body, err := g.buildStmts(f.Body)
		if err != nil {
			return nil, err
		}
		g.popScope()

		forStmt = GoForStmt{
			Init: fmt.Sprintf("rugo_range_i, rugo_range_idx := %s, 0", startExpr),
			Cond: fmt.Sprintf("rugo_range_i < %s", endExpr),
			Post: "rugo_range_i, rugo_range_idx = rugo_range_i+1, rugo_range_idx+1",
			Body: append(preamble, body...),
		}
	} else {
		// Single-variable form
		preamble = append(preamble, GoAssignStmt{Target: iterVar, Op: ":=", Value: GoRawExpr{Code: "rugo_range_i"}})
		preamble = append(preamble, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", iterVar)}})
		g.declareVar(iterVar)

		body, err := g.buildStmts(f.Body)
		if err != nil {
			return nil, err
		}
		g.popScope()

		forStmt = GoForStmt{
			Init: fmt.Sprintf("rugo_range_i := %s", startExpr),
			Cond: fmt.Sprintf("rugo_range_i < %s", endExpr),
			Post: "rugo_range_i++",
			Body: append(preamble, body...),
		}
	}

	return []GoStmt{forStmt}, nil
}

// --- Function builder ---

// buildFunc converts a Rugo FuncDef into a GoFuncDecl.
func (g *codeGen) buildFunc(f *ast.FuncDef) (GoFuncDecl, error) {
	if f.SourceFile != "" {
		saved := g.sourceFile
		g.sourceFile = f.SourceFile
		g.w.source = f.SourceFile
		defer func() { g.sourceFile = saved; g.w.source = saved }()
	}

	hasDefaults := ast.HasDefaults(f.Params)
	fti := g.funcTypeInfo(f)

	// Determine Go function name
	var goName string
	if f.Namespace != "" {
		goName = fmt.Sprintf("rugons_%s_%s", f.Namespace, f.Name)
	} else {
		goName = fmt.Sprintf("rugofn_%s", f.Name)
	}

	retType := "interface{}"
	if fti != nil && fti.ReturnType.IsTyped() {
		retType = fti.ReturnType.GoType()
	}

	// Build params
	var params []GoParam
	if hasDefaults {
		params = []GoParam{{Name: "_args", Type: "...interface{}"}}
	} else {
		params = make([]GoParam, len(f.Params))
		for i, p := range f.Params {
			if fti != nil && fti.ParamTypes[i].IsTyped() {
				params[i] = GoParam{Name: p.Name, Type: fti.ParamTypes[i].GoType()}
			} else {
				params[i] = GoParam{Name: p.Name, Type: "interface{}"}
			}
		}
	}

	// Build body
	g.pushScope()
	var body []GoStmt

	// Recursion depth guard
	body = append(body, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("rugo_check_depth(%q)", f.Name)}})
	body = append(body, GoRawStmt{Code: "defer func() { rugo_call_depth-- }()"})

	if hasDefaults {
		// Arity range check
		minArity := ast.MinArity(f.Params)
		maxArity := len(f.Params)
		body = append(body, GoRawStmt{Code: fmt.Sprintf(
			"if len(_args) < %d { panic(fmt.Sprintf(\"%s() takes %d to %d arguments but %%d were given\", len(_args))) }",
			minArity, f.Name, minArity, maxArity)})
		body = append(body, GoRawStmt{Code: fmt.Sprintf(
			"if len(_args) > %d { panic(fmt.Sprintf(\"%s() takes %d to %d arguments but %%d were given\", len(_args))) }",
			maxArity, f.Name, minArity, maxArity)})

		// Unpack params
		for i, p := range f.Params {
			g.declareVar(p.Name)
			if p.Default == nil {
				body = append(body, GoRawStmt{Code: fmt.Sprintf("var %s interface{} = _args[%d]", p.Name, i)})
			} else {
				defaultExpr, err := g.exprString(p.Default)
				if err != nil {
					return GoFuncDecl{}, err
				}
				body = append(body, GoRawStmt{Code: fmt.Sprintf("var %s interface{}", p.Name)})
				body = append(body, GoRawStmt{Code: fmt.Sprintf("if len(_args) > %d { %s = _args[%d] } else { %s = %s }", i, p.Name, i, p.Name, defaultExpr)})
			}
			body = append(body, GoExprStmt{Expr: GoRawExpr{Code: fmt.Sprintf("_ = %s", p.Name)}})
		}
	} else {
		for _, p := range f.Params {
			g.declareVar(p.Name)
		}
	}

	g.currentFunc = f
	g.inFunc = true
	bodyStmts, err := g.buildStmts(f.Body)
	if err != nil {
		g.inFunc = false
		g.currentFunc = nil
		g.popScope()
		return GoFuncDecl{}, err
	}
	body = append(body, bodyStmts...)

	if !g.bodyHasImplicitReturn(f.Body) {
		if fti != nil && fti.ReturnType.IsTyped() {
			body = append(body, GoReturnStmt{Value: GoRawExpr{Code: typedZero(fti.ReturnType)}})
		} else {
			body = append(body, GoReturnStmt{Value: GoRawExpr{Code: "nil"}})
		}
	}
	g.inFunc = false
	g.currentFunc = nil
	g.popScope()

	return GoFuncDecl{
		Name:   goName,
		Params: params,
		Return: retType,
		Body:   body,
	}, nil
}
