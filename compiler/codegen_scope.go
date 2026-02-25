package compiler

import (
	"github.com/rubiojr/rugo/ast"
	"github.com/rubiojr/rugo/preprocess"
	"sort"

	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
)

// collectIdents returns the set of identifier names referenced in statements.
func collectIdents(stmts []ast.Statement) map[string]bool {
	names := make(map[string]bool)
	for _, s := range stmts {
		collectIdentsFromStmt(s, names)
	}
	return names
}

func collectIdentsFromStmt(s ast.Statement, names map[string]bool) {
	switch st := s.(type) {
	case *ast.AssignStmt:
		collectIdentsFromExpr(st.Value, names)
	case *ast.IndexAssignStmt:
		collectIdentsFromExpr(st.Object, names)
		collectIdentsFromExpr(st.Index, names)
		collectIdentsFromExpr(st.Value, names)
	case *ast.DotAssignStmt:
		collectIdentsFromExpr(st.Object, names)
		collectIdentsFromExpr(st.Value, names)
	case *ast.ExprStmt:
		collectIdentsFromExpr(st.Expression, names)
	case *ast.IfStmt:
		collectIdentsFromExpr(st.Condition, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
		for _, ei := range st.ElsifClauses {
			collectIdentsFromExpr(ei.Condition, names)
			for _, b := range ei.Body {
				collectIdentsFromStmt(b, names)
			}
		}
		for _, b := range st.ElseBody {
			collectIdentsFromStmt(b, names)
		}
	case *ast.CaseStmt:
		collectIdentsFromExpr(st.Subject, names)
		for _, oc := range st.OfClauses {
			for _, v := range oc.Values {
				collectIdentsFromExpr(v, names)
			}
			if oc.ArrowExpr != nil {
				collectIdentsFromExpr(oc.ArrowExpr, names)
			}
			for _, b := range oc.Body {
				collectIdentsFromStmt(b, names)
			}
		}
		for _, ei := range st.ElsifClauses {
			collectIdentsFromExpr(ei.Condition, names)
			for _, b := range ei.Body {
				collectIdentsFromStmt(b, names)
			}
		}
		for _, b := range st.ElseBody {
			collectIdentsFromStmt(b, names)
		}
	case *ast.WhileStmt:
		collectIdentsFromExpr(st.Condition, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.ForStmt:
		collectIdentsFromExpr(st.Collection, names)
		for _, b := range st.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.ReturnStmt:
		if st.Value != nil {
			collectIdentsFromExpr(st.Value, names)
		}
	case *ast.ImplicitReturnStmt:
		collectIdentsFromExpr(st.Value, names)
	case *ast.TryResultStmt:
		collectIdentsFromExpr(st.Value, names)
	case *ast.SpawnReturnStmt:
		if st.Value != nil {
			collectIdentsFromExpr(st.Value, names)
		}
	case *ast.TryHandlerReturnStmt:
		if st.Value != nil {
			collectIdentsFromExpr(st.Value, names)
		}
	}
}

func collectIdentsFromExpr(e ast.Expr, names map[string]bool) {
	switch ex := e.(type) {
	case *ast.IdentExpr:
		names[ex.Name] = true
	case *ast.BinaryExpr:
		collectIdentsFromExpr(ex.Left, names)
		collectIdentsFromExpr(ex.Right, names)
	case *ast.UnaryExpr:
		collectIdentsFromExpr(ex.Operand, names)
	case *ast.CallExpr:
		collectIdentsFromExpr(ex.Func, names)
		for _, a := range ex.Args {
			collectIdentsFromExpr(a, names)
		}
	case *ast.IndexExpr:
		collectIdentsFromExpr(ex.Object, names)
		collectIdentsFromExpr(ex.Index, names)
	case *ast.SliceExpr:
		collectIdentsFromExpr(ex.Object, names)
		collectIdentsFromExpr(ex.Start, names)
		collectIdentsFromExpr(ex.Length, names)
	case *ast.DotExpr:
		collectIdentsFromExpr(ex.Object, names)
	case *ast.ArrayLiteral:
		for _, el := range ex.Elements {
			collectIdentsFromExpr(el, names)
		}
	case *ast.HashLiteral:
		for _, p := range ex.Pairs {
			collectIdentsFromExpr(p.Key, names)
			collectIdentsFromExpr(p.Value, names)
		}
	case *ast.LoweredTryExpr:
		collectIdentsFromExpr(ex.Expr, names)
		for _, b := range ex.Handler {
			collectIdentsFromStmt(b, names)
		}
		if ex.ResultExpr != nil {
			collectIdentsFromExpr(ex.ResultExpr, names)
		}
	case *ast.LoweredSpawnExpr:
		for _, s := range ex.Body {
			collectIdentsFromStmt(s, names)
		}
		if ex.ResultExpr != nil {
			collectIdentsFromExpr(ex.ResultExpr, names)
		}
	case *ast.LoweredParallelExpr:
		for _, br := range ex.Branches {
			if br.Expr != nil {
				collectIdentsFromExpr(br.Expr, names)
			}
			for _, s := range br.Stmts {
				collectIdentsFromStmt(s, names)
			}
		}
	case *ast.CaseExpr:
		collectIdentsFromExpr(ex.Subject, names)
		for _, oc := range ex.OfClauses {
			for _, v := range oc.Values {
				collectIdentsFromExpr(v, names)
			}
			if oc.ArrowExpr != nil {
				collectIdentsFromExpr(oc.ArrowExpr, names)
			}
			for _, s := range oc.Body {
				collectIdentsFromStmt(s, names)
			}
		}
		for _, ec := range ex.ElsifClauses {
			collectIdentsFromExpr(ec.Condition, names)
			for _, s := range ec.Body {
				collectIdentsFromStmt(s, names)
			}
		}
		for _, s := range ex.ElseBody {
			collectIdentsFromStmt(s, names)
		}
	case *ast.FnExpr:
		for _, b := range ex.Body {
			collectIdentsFromStmt(b, names)
		}
	case *ast.StringLiteral:
		if preprocess.HasInterpolation(ex.Value) {
			_, exprStrs, err := preprocess.ProcessInterpolation(ex.Value)
			if err != nil {
				break
			}
			for _, exprStr := range exprStrs {
				p := &parser.Parser{}
				flatAST, err := p.Parse("<ident-scan>", []byte(exprStr+"\n"))
				if err != nil {
					continue
				}
				prog, err := ast.Walk(p, flatAST)
				if err != nil {
					continue
				}
				for _, s := range prog.Statements {
					collectIdentsFromStmt(s, names)
				}
			}
		}
	}
}

// Scope management
func (g *codeGen) pushScope() {
	g.scopes = append(g.scopes, make(map[string]bool))
	g.constScopes = append(g.constScopes, make(map[string]int))
}

func (g *codeGen) popScope() {
	g.scopes = g.scopes[:len(g.scopes)-1]
	g.constScopes = g.constScopes[:len(g.constScopes)-1]
}

func (g *codeGen) declareVar(name string) {
	g.scopes[len(g.scopes)-1][name] = true
}

func (g *codeGen) isDeclared(name string) bool {
	for i := len(g.scopes) - 1; i >= 0; i-- {
		if g.scopes[i][name] {
			return true
		}
	}
	return false
}

// isCapturedVar returns true if name is declared in an outer scope beyond
// the current lambda boundary (i.e., it's a captured variable from the
// enclosing scope, not a local variable of the lambda).
func (g *codeGen) isCapturedVar(name string) bool {
	if g.lambdaDepth == 0 {
		return false
	}
	base := g.lambdaScopeBase[len(g.lambdaScopeBase)-1]
	// Check if declared in scopes below the lambda boundary
	for i := len(g.scopes) - 1; i >= base; i-- {
		if g.scopes[i][name] {
			return false // declared inside the lambda
		}
	}
	for i := base - 1; i >= 0; i-- {
		if g.scopes[i][name] {
			return true // declared outside the lambda
		}
	}
	return false
}

// capturedVarType returns the inferred type of a captured variable
// by looking it up in the enclosing function's scope.
func (g *codeGen) capturedVarType(name string) RugoType {
	if g.typeInfo == nil || g.lambdaDepth == 0 {
		return TypeDynamic
	}
	outerFunc := g.lambdaOuterFunc[len(g.lambdaOuterFunc)-1]
	scope := ""
	if outerFunc != nil {
		scope = funcKey(outerFunc)
	} else if g.varTypeScope != "" {
		scope = g.varTypeScope
	}
	return g.typeInfo.VarType(scope, name)
}

func (g *codeGen) declareConst(name string, line int) {
	g.constScopes[len(g.constScopes)-1][name] = line
}

func (g *codeGen) constantLine(name string) (int, bool) {
	for i := len(g.constScopes) - 1; i >= 0; i-- {
		if line, ok := g.constScopes[i][name]; ok {
			return line, true
		}
	}
	return 0, false
}

// importedModuleNames returns sorted module names from the imports map.
func importedModuleNames(imports map[string]bool) []string {
	var names []string
	for name := range imports {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// astUsesSpawn checks if any LoweredSpawnExpr exists in the AST.
func astUsesSpawn(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		_, ok := e.(*ast.LoweredSpawnExpr)
		return ok
	})
}

// astUsesTaskMethods checks if any ast.DotExpr uses .value, .done, or .wait on a non-module target.
func astUsesTaskMethods(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		dot, ok := e.(*ast.DotExpr)
		if !ok || !taskMethodNames[dot.Field] {
			return false
		}
		if ident, ok := dot.Object.(*ast.IdentExpr); ok {
			if modules.IsModule(ident.Name) {
				return false
			}
		}
		return true
	})
}

var taskMethodNames = map[string]bool{"value": true, "done": true, "wait": true}

// astUsesParallel checks if any LoweredParallelExpr exists in the AST.
func astUsesParallel(prog *ast.Program) bool {
	return WalkExprs(prog, func(e ast.Expr) bool {
		_, ok := e.(*ast.LoweredParallelExpr)
		return ok
	})
}
