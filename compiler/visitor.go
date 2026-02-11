package compiler

import "github.com/rubiojr/rugo/ast"

// WalkExprs traverses the entire AST and calls fn on every ast.Expr node.
// Returns true as soon as fn returns true for any expression.
func WalkExprs(prog *ast.Program, fn func(ast.Expr) bool) bool {
	for _, s := range prog.Statements {
		if walkStmtExprs(s, fn) {
			return true
		}
	}
	return false
}

// WalkStmts traverses all statements in a ast.Program, calling fn for each.
// If fn returns false, children of that statement are not visited.
// Recurses into block bodies (ast.FuncDef, ast.TestDef, ast.BenchDef, ast.IfStmt, ast.WhileStmt, ast.ForStmt).
func WalkStmts(prog *ast.Program, fn func(ast.Statement) bool) {
	for _, s := range prog.Statements {
		walkStmtRecursive(s, fn)
	}
}

func walkStmtRecursive(s ast.Statement, fn func(ast.Statement) bool) {
	if !fn(s) {
		return
	}
	switch st := s.(type) {
	case *ast.FuncDef:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
	case *ast.TestDef:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
	case *ast.BenchDef:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
	case *ast.IfStmt:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
		for _, clause := range st.ElsifClauses {
			for _, child := range clause.Body {
				walkStmtRecursive(child, fn)
			}
		}
		for _, child := range st.ElseBody {
			walkStmtRecursive(child, fn)
		}
	case *ast.WhileStmt:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
	case *ast.ForStmt:
		for _, child := range st.Body {
			walkStmtRecursive(child, fn)
		}
	}
}

func walkStmtExprs(s ast.Statement, fn func(ast.Expr) bool) bool {
	switch st := s.(type) {
	case *ast.ExprStmt:
		return walkExpr(st.Expression, fn)
	case *ast.AssignStmt:
		return walkExpr(st.Value, fn)
	case *ast.IndexAssignStmt:
		return walkExpr(st.Object, fn) || walkExpr(st.Index, fn) || walkExpr(st.Value, fn)
	case *ast.DotAssignStmt:
		return walkExpr(st.Object, fn) || walkExpr(st.Value, fn)
	case *ast.IfStmt:
		if walkExpr(st.Condition, fn) {
			return true
		}
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
		for _, ec := range st.ElsifClauses {
			if walkExpr(ec.Condition, fn) {
				return true
			}
			for _, s := range ec.Body {
				if walkStmtExprs(s, fn) {
					return true
				}
			}
		}
		for _, s := range st.ElseBody {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.WhileStmt:
		if walkExpr(st.Condition, fn) {
			return true
		}
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.ForStmt:
		if walkExpr(st.Collection, fn) {
			return true
		}
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.FuncDef:
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.TestDef:
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.BenchDef:
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.ReturnStmt:
		if st.Value != nil {
			return walkExpr(st.Value, fn)
		}
	}
	return false
}

// walkExpr calls fn on the expression, then recurses into child expressions.
// Returns true as soon as fn returns true.
func walkExpr(e ast.Expr, fn func(ast.Expr) bool) bool {
	if e == nil {
		return false
	}
	if fn(e) {
		return true
	}
	switch ex := e.(type) {
	case *ast.CallExpr:
		if walkExpr(ex.Func, fn) {
			return true
		}
		for _, a := range ex.Args {
			if walkExpr(a, fn) {
				return true
			}
		}
	case *ast.BinaryExpr:
		return walkExpr(ex.Left, fn) || walkExpr(ex.Right, fn)
	case *ast.UnaryExpr:
		return walkExpr(ex.Operand, fn)
	case *ast.IndexExpr:
		return walkExpr(ex.Object, fn) || walkExpr(ex.Index, fn)
	case *ast.DotExpr:
		return walkExpr(ex.Object, fn)
	case *ast.ArrayLiteral:
		for _, el := range ex.Elements {
			if walkExpr(el, fn) {
				return true
			}
		}
	case *ast.HashLiteral:
		for _, p := range ex.Pairs {
			if walkExpr(p.Key, fn) || walkExpr(p.Value, fn) {
				return true
			}
		}
	case *ast.TryExpr:
		if walkExpr(ex.Expr, fn) {
			return true
		}
		for _, s := range ex.Handler {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.SpawnExpr:
		for _, s := range ex.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ast.ParallelExpr:
		for _, s := range ex.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	}
	return false
}
