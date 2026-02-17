package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"
)

func (g *codeGen) writeStmt(s ast.Statement) error {
	nodes, err := g.buildStmt(s)
	if err != nil {
		return err
	}
	g.emitGoStmts(nodes)
	return nil
}

// emitGoStmts writes Go output AST nodes through the old goWriter.
// This is the bridge between the new buildStmt and the old write system.
func (g *codeGen) emitGoStmts(nodes []GoStmt) {
	for _, n := range nodes {
		g.emitGoStmt(n)
	}
}

func (g *codeGen) emitGoStmt(s GoStmt) {
	switch st := s.(type) {
	case GoExprStmt:
		g.writef("%s\n", g.goExprStr(st.Expr))
	case GoAssignStmt:
		g.writef("%s %s %s\n", st.Target, st.Op, g.goExprStr(st.Value))
	case GoMultiAssignStmt:
		g.writef("%s %s %s\n", strings.Join(st.Targets, ", "), st.Op, g.goExprStr(st.Value))
	case GoReturnStmt:
		if st.Value != nil {
			g.writef("return %s\n", g.goExprStr(st.Value))
		} else {
			g.writeln("return")
		}
	case GoIfStmt:
		g.writef("if %s {\n", g.goExprStr(st.Cond))
		g.w.Indent()
		g.emitGoStmts(st.Body)
		g.w.Dedent()
		for _, ei := range st.ElseIf {
			g.writef("} else if %s {\n", g.goExprStr(ei.Cond))
			g.w.Indent()
			g.emitGoStmts(ei.Body)
			g.w.Dedent()
		}
		if len(st.Else) > 0 {
			g.writeln("} else {")
			g.w.Indent()
			g.emitGoStmts(st.Else)
			g.w.Dedent()
		}
		g.writeln("}")
	case GoForStmt:
		if st.Init != "" {
			g.writef("for %s; %s; %s {\n", st.Init, st.Cond, st.Post)
		} else if st.Cond != "" {
			g.writef("for %s {\n", st.Cond)
		} else {
			g.writeln("for {")
		}
		g.w.Indent()
		g.emitGoStmts(st.Body)
		g.w.Dedent()
		g.writeln("}")
	case GoForRangeStmt:
		if st.Value != "" {
			g.writef("for %s, %s := range %s {\n", st.Key, st.Value, g.goExprStr(st.Collection))
		} else {
			g.writef("for %s := range %s {\n", st.Key, g.goExprStr(st.Collection))
		}
		g.w.Indent()
		g.emitGoStmts(st.Body)
		g.w.Dedent()
		g.writeln("}")
	case GoDeferStmt:
		g.writeln("defer func() {")
		g.w.Indent()
		g.emitGoStmts(st.Body)
		g.w.Dedent()
		g.writeln("}()")
	case GoGoStmt:
		g.writeln("go func() {")
		g.w.Indent()
		g.emitGoStmts(st.Body)
		g.w.Dedent()
		g.writeln("}()")
	case GoBreakStmt:
		g.writeln("break")
	case GoContinueStmt:
		g.writeln("continue")
	case GoBlankLine:
		g.writeln("")
	case GoLineDirective:
		if st.Line > 0 && st.File != "" {
			g.w.sb.WriteString(fmt.Sprintf("//line %s:%d\n", st.File, st.Line))
		}
	case GoComment:
		g.writef("// %s\n", st.Text)
	case GoRawStmt:
		g.writef("%s\n", st.Code)
	}
}

func (g *codeGen) goExprStr(e GoExpr) string {
	switch ex := e.(type) {
	case GoRawExpr:
		return ex.Code
	default:
		return "<unknown>"
	}
}

// bodyHasImplicitReturn checks if all code paths in a body produce a value
// via ImplicitReturnStmt nodes. Returns true if no default return is needed.
func (g *codeGen) bodyHasImplicitReturn(body []ast.Statement) bool {
	if len(body) == 0 {
		return false
	}
	return stmtCoversAllPaths(body[len(body)-1])
}

// stmtCoversAllPaths returns true if a statement produces a return value
// in all code paths (ImplicitReturnStmt always does; IfStmt does if all
// branches including else cover all paths).
func stmtCoversAllPaths(s ast.Statement) bool {
	switch st := s.(type) {
	case *ast.ImplicitReturnStmt:
		return true
	case *ast.IfStmt:
		if len(st.ElseBody) == 0 {
			return false
		}
		if !branchCoversAllPaths(st.Body) {
			return false
		}
		for _, ec := range st.ElsifClauses {
			if !branchCoversAllPaths(ec.Body) {
				return false
			}
		}
		return branchCoversAllPaths(st.ElseBody)
	default:
		return false
	}
}

func branchCoversAllPaths(body []ast.Statement) bool {
	if len(body) == 0 {
		return false
	}
	return stmtCoversAllPaths(body[len(body)-1])
}

// stmtError wraps a codegen error with file:line context from the statement.
func (g *codeGen) stmtError(s ast.Statement, err error) error {
	line := s.StmtLine()
	msg := err.Error()
	// Strip existing "line N: " prefix if present
	if strings.HasPrefix(msg, "line ") {
		if idx := strings.Index(msg, ": "); idx != -1 {
			msg = msg[idx+2:]
		}
	}
	// Strip existing "file:N: " prefix if present (from nested stmtError)
	if g.sourceFile != "" && strings.HasPrefix(msg, g.sourceFile+":") {
		rest := msg[len(g.sourceFile)+1:]
		if idx := strings.Index(rest, ": "); idx != -1 {
			msg = rest[idx+2:]
		}
	}
	if line > 0 && g.sourceFile != "" {
		return fmt.Errorf("%s:%d: %s", g.sourceFile, line, msg)
	}
	return err
}

func (g *codeGen) writeAssign(a *ast.AssignStmt) error {
	// Uppercase names are constants — reject reassignment
	if origLine, ok := g.constantLine(a.Target); ok {
		return fmt.Errorf("cannot reassign constant %s (first assigned at line %d)", a.Target, origLine)
	}

	exprType := g.exprType(a.Value)
	varType := g.varType(a.Target)

	// For captured variables inside lambdas, use the outer scope's type
	// to ensure proper type conversion in the generated Go code.
	isCaptured := g.isCapturedVar(a.Target)
	if isCaptured {
		varType = g.capturedVarType(a.Target)
	}

	// If the variable is dynamic but the expression is typed, box the value.
	expr, err := g.exprString(a.Value)
	if err != nil {
		return err
	}
	if !varType.IsTyped() && exprType.IsTyped() {
		expr = fmt.Sprintf("interface{}(%s)", expr)
	}
	// If the variable is typed but the expression is dynamic, add a type assertion.
	// This happens when a captured variable (declared typed in outer scope)
	// is assigned a dynamic expression inside a lambda.
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

	if g.isDeclared(a.Target) || (g.handlerVars[a.Target] && !g.inFunc) {
		g.writef("%s = %s\n", a.Target, expr)
		// Track constants for handler vars on first use in main()
		if g.handlerVars[a.Target] && !g.isDeclared(a.Target) {
			g.declareVar(a.Target)
			if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
				g.declareConst(a.Target, a.SourceLine)
			}
		}
	} else {
		g.writef("%s := %s\n", a.Target, expr)
		g.declareVar(a.Target)
		if len(a.Target) > 0 && a.Target[0] >= 'A' && a.Target[0] <= 'Z' {
			g.declareConst(a.Target, a.SourceLine)
		}
	}
	// Suppress "declared but not used" by referencing with _
	g.writef("_ = %s\n", a.Target)
	return nil
}

func (g *codeGen) writeIndexAssign(ia *ast.IndexAssignStmt) error {
	obj, err := g.exprString(ia.Object)
	if err != nil {
		return err
	}
	idx, err := g.exprString(ia.Index)
	if err != nil {
		return err
	}
	val, err := g.exprString(ia.Value)
	if err != nil {
		return err
	}
	g.writef("rugo_index_set(%s, %s, %s)\n", obj, idx, val)
	return nil
}

func (g *codeGen) writeDotAssign(da *ast.DotAssignStmt) error {
	if da.Field == "__type__" {
		return fmt.Errorf("cannot assign to .__type__ — use type_of() for type introspection")
	}
	obj, err := g.exprString(da.Object)
	if err != nil {
		return err
	}
	val, err := g.exprString(da.Value)
	if err != nil {
		return err
	}
	g.writef("rugo_dot_set(%s, %q, %s)\n", obj, da.Field, val)
	return nil
}

func (g *codeGen) writeExprStmt(e *ast.ExprStmt) error {
	expr, err := g.exprString(e.Expression)
	if err != nil {
		return err
	}
	g.writef("_ = %s\n", expr)
	return nil
}

func (g *codeGen) writeIf(i *ast.IfStmt) error {
	// Pre-declare variables assigned in any branch so they're visible
	// after the if block (Ruby-like scoping: if/else doesn't create a new scope).
	g.predeclareIfVars(i)

	cond, err := g.exprString(i.Condition)
	if err != nil {
		return err
	}
	g.writef("if %s {\n", g.condExpr(cond, i.Condition))
	g.w.Indent()
	for _, s := range i.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.w.Dedent()
	for _, ec := range i.ElsifClauses {
		cond, err := g.exprString(ec.Condition)
		if err != nil {
			return err
		}
		g.writef("} else if %s {\n", g.condExpr(cond, ec.Condition))
		g.w.Indent()
		for _, s := range ec.Body {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.w.Dedent()
	}
	if len(i.ElseBody) > 0 {
		g.writeln("} else {")
		g.w.Indent()
		for _, s := range i.ElseBody {
			if err := g.writeStmt(s); err != nil {
				return err
			}
		}
		g.w.Dedent()
	}
	g.writeln("}")
	return nil
}

// predeclareIfVars pre-declares variables assigned in any branch of an if/else
// so they're visible after the if block (Ruby-like scoping).
func (g *codeGen) predeclareIfVars(i *ast.IfStmt) {
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
				g.writef("var %s %s\n", name, varType.GoType())
			} else {
				g.writef("var %s interface{}\n", name)
			}
			g.declareVar(name)
		}
	}
}

// collectAssignTargets returns variable names assigned in a list of statements,
// in order of first appearance. It recurses into nested if/else blocks.
func collectAssignTargets(stmts []ast.Statement) []string {
	var names []string
	seen := make(map[string]bool)
	var collect func([]ast.Statement)
	collect = func(stmts []ast.Statement) {
		for _, s := range stmts {
			switch st := s.(type) {
			case *ast.AssignStmt:
				if !seen[st.Target] {
					names = append(names, st.Target)
					seen[st.Target] = true
				}
			case *ast.IfStmt:
				collect(st.Body)
				for _, clause := range st.ElsifClauses {
					collect(clause.Body)
				}
				collect(st.ElseBody)
			}
		}
	}
	collect(stmts)
	return names
}

func (g *codeGen) writeWhile(w *ast.WhileStmt) error {
	cond, err := g.exprString(w.Condition)
	if err != nil {
		return err
	}
	g.writef("for %s {\n", g.condExpr(cond, w.Condition))
	g.w.Indent()
	g.pushScope()
	for _, s := range w.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.popScope()
	g.w.Dedent()
	g.writeln("}")
	return nil
}

func (g *codeGen) writeFor(f *ast.ForStmt) error {
	// Try to emit an optimized integer for-loop (no slice allocation).
	if g.writeForRange(f) {
		return g.writeForBody(f)
	}

	coll, err := g.exprString(f.Collection)
	if err != nil {
		return err
	}

	iterVar := f.Var
	idxVar := f.IndexVar

	// Declare the loop variable(s)
	if idxVar != "" {
		// Two-variable form: for key, val in hash / for idx, val in arr
		g.writef("for _, rugo_for_kv := range rugo_iterable(%s) {\n", coll)
		g.w.Indent()
		g.pushScope()
		if iterVar == "_" {
			g.writef("_ = rugo_for_kv.Key\n")
		} else {
			g.writef("%s := rugo_for_kv.Key\n", iterVar)
			g.writef("_ = %s\n", iterVar)
			g.declareVar(iterVar)
		}
		if idxVar == "_" {
			g.writef("_ = rugo_for_kv.Val\n")
		} else {
			g.writef("%s := rugo_for_kv.Val\n", idxVar)
			g.writef("_ = %s\n", idxVar)
			g.declareVar(idxVar)
		}
	} else {
		// Single-variable form: for val in arr / for key in hash
		g.writef("for _, rugo_for_item := range rugo_iterable_default(%s) {\n", coll)
		g.w.Indent()
		g.pushScope()
		g.writef("%s := rugo_for_item\n", iterVar)
		g.writef("_ = %s\n", iterVar)
		g.declareVar(iterVar)
	}

	return g.writeForBody(f)
}

// writeForBody writes the loop body, pops scope, and closes the block.
func (g *codeGen) writeForBody(f *ast.ForStmt) error {
	for _, s := range f.Body {
		if err := g.writeStmt(s); err != nil {
			return err
		}
	}
	g.popScope()
	g.w.Dedent()
	g.writeln("}")
	return nil
}

// writeForRange detects range() calls and integer literals in for-loop
// collections and emits an efficient Go for-loop (no slice allocation).
// Returns true if the optimization was applied.
func (g *codeGen) writeForRange(f *ast.ForStmt) bool {
	startExpr, endExpr := g.rangeExprs(f.Collection)
	if startExpr == "" {
		return false
	}

	iterVar := f.Var
	idxVar := f.IndexVar

	if idxVar != "" {
		// Two-variable form: for idx, val in range(5, 20)
		g.writef("for rugo_range_i, rugo_range_idx := %s, 0; rugo_range_i < %s; rugo_range_i, rugo_range_idx = rugo_range_i+1, rugo_range_idx+1 {\n", startExpr, endExpr)
		g.w.Indent()
		g.pushScope()
		if iterVar == "_" {
			g.writef("_ = rugo_range_idx\n")
		} else {
			g.writef("%s := rugo_range_idx\n", iterVar)
			g.writef("_ = %s\n", iterVar)
			g.declareVar(iterVar)
		}
		if idxVar == "_" {
			g.writef("_ = rugo_range_i\n")
		} else {
			g.writef("%s := rugo_range_i\n", idxVar)
			g.writef("_ = %s\n", idxVar)
			g.declareVar(idxVar)
		}
	} else {
		// Single-variable form: for i in range(5, 20)
		g.writef("for rugo_range_i := %s; rugo_range_i < %s; rugo_range_i++ {\n", startExpr, endExpr)
		g.w.Indent()
		g.pushScope()
		g.writef("%s := rugo_range_i\n", iterVar)
		g.writef("_ = %s\n", iterVar)
		g.declareVar(iterVar)
	}
	return true
}

// rangeExprs detects optimizable range patterns in the collection expression.
// Returns (startExpr, endExpr) Go code strings, or ("", "") if not optimizable.
func (g *codeGen) rangeExprs(coll ast.Expr) (string, string) {
	// for i in 100 (integer literal)
	if intLit, ok := coll.(*ast.IntLiteral); ok {
		return "0", intLit.Value
	}

	// for i in n (integer variable)
	if ident, ok := coll.(*ast.IdentExpr); ok {
		if g.varType(ident.Name) == TypeInt {
			return "0", ident.Name
		}
	}

	// for i in range(...)
	call, ok := coll.(*ast.CallExpr)
	if !ok {
		return "", ""
	}
	ident, ok := call.Func.(*ast.IdentExpr)
	if !ok || ident.Name != "range" {
		return "", ""
	}

	switch len(call.Args) {
	case 1:
		return "0", g.rangeIntExpr(call.Args[0])
	case 2:
		return g.rangeIntExpr(call.Args[0]), g.rangeIntExpr(call.Args[1])
	}
	return "", ""
}

// rangeIntExpr returns a Go int expression for a range bound.
// Integer literals pass through directly; other expressions are
// wrapped in rugo_to_int() for runtime conversion.
func (g *codeGen) rangeIntExpr(e ast.Expr) string {
	if intLit, ok := e.(*ast.IntLiteral); ok {
		return intLit.Value
	}
	s, err := g.exprString(e)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("rugo_to_int(%s)", s)
}

func (g *codeGen) writeReturn(r *ast.ReturnStmt) error {
	fti := g.currentFuncTypeInfo()
	if r.Value == nil {
		if fti != nil && fti.ReturnType.IsTyped() {
			g.writef("return %s\n", typedZero(fti.ReturnType))
		} else {
			g.writeln("return nil")
		}
	} else {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return err
		}
		g.writef("return %s\n", expr)
	}
	return nil
}

func (g *codeGen) writeImplicitReturn(r *ast.ImplicitReturnStmt) error {
	expr, err := g.exprString(r.Value)
	if err != nil {
		return err
	}
	g.writef("return %s\n", expr)
	return nil
}

func (g *codeGen) writeTryResult(r *ast.TryResultStmt) error {
	expr, err := g.exprString(r.Value)
	if err != nil {
		return err
	}
	g.writef("r = %s\n", expr)
	return nil
}

func (g *codeGen) writeSpawnReturn(r *ast.SpawnReturnStmt) error {
	if r.Value != nil {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return err
		}
		g.writef("t.result = %s\n", expr)
	}
	g.writeln("return")
	return nil
}

func (g *codeGen) writeTryHandlerReturn(r *ast.TryHandlerReturnStmt) error {
	if r.Value != nil {
		expr, err := g.exprString(r.Value)
		if err != nil {
			return err
		}
		g.writef("r = %s\n", expr)
	}
	g.writeln("return")
	return nil
}
