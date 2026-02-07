package compiler

// walkExpressions traverses the entire AST and calls fn on every Expr node.
// Returns true as soon as fn returns true for any expression.
func walkExpressions(prog *Program, fn func(Expr) bool) bool {
	for _, s := range prog.Statements {
		if walkStmtExprs(s, fn) {
			return true
		}
	}
	return false
}

func walkStmtExprs(s Statement, fn func(Expr) bool) bool {
	switch st := s.(type) {
	case *ExprStmt:
		return walkExpr(st.Expression, fn)
	case *AssignStmt:
		return walkExpr(st.Value, fn)
	case *IndexAssignStmt:
		return walkExpr(st.Object, fn) || walkExpr(st.Index, fn) || walkExpr(st.Value, fn)
	case *IfStmt:
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
	case *WhileStmt:
		if walkExpr(st.Condition, fn) {
			return true
		}
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ForStmt:
		if walkExpr(st.Collection, fn) {
			return true
		}
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *FuncDef:
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *TestDef:
		for _, s := range st.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ReturnStmt:
		if st.Value != nil {
			return walkExpr(st.Value, fn)
		}
	}
	return false
}

// walkExpr calls fn on the expression, then recurses into child expressions.
// Returns true as soon as fn returns true.
func walkExpr(e Expr, fn func(Expr) bool) bool {
	if e == nil {
		return false
	}
	if fn(e) {
		return true
	}
	switch ex := e.(type) {
	case *CallExpr:
		if walkExpr(ex.Func, fn) {
			return true
		}
		for _, a := range ex.Args {
			if walkExpr(a, fn) {
				return true
			}
		}
	case *BinaryExpr:
		return walkExpr(ex.Left, fn) || walkExpr(ex.Right, fn)
	case *UnaryExpr:
		return walkExpr(ex.Operand, fn)
	case *IndexExpr:
		return walkExpr(ex.Object, fn) || walkExpr(ex.Index, fn)
	case *DotExpr:
		return walkExpr(ex.Object, fn)
	case *ArrayLiteral:
		for _, el := range ex.Elements {
			if walkExpr(el, fn) {
				return true
			}
		}
	case *HashLiteral:
		for _, p := range ex.Pairs {
			if walkExpr(p.Key, fn) || walkExpr(p.Value, fn) {
				return true
			}
		}
	case *TryExpr:
		if walkExpr(ex.Expr, fn) {
			return true
		}
		for _, s := range ex.Handler {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *SpawnExpr:
		for _, s := range ex.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	case *ParallelExpr:
		for _, s := range ex.Body {
			if walkStmtExprs(s, fn) {
				return true
			}
		}
	}
	return false
}
