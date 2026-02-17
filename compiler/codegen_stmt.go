package compiler

import (
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"strings"
)

func (g *codeGen) goExprStr(e GoExpr) string {
	switch ex := e.(type) {
	case GoRawExpr:
		return ex.Code
	case GoIIFEExpr:
		return g.renderIIFE(ex)
	default:
		return "<unknown>"
	}
}

// renderIIFE renders a GoIIFEExpr to a string at the current indent level.
func (g *codeGen) renderIIFE(e GoIIFEExpr) string {
	p := &goPrinter{indent: g.indent + 1}
	retType := e.ReturnType
	if retType == "" {
		retType = "interface{}"
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "func() %s {\n", retType)
	for _, s := range e.Body {
		p.printStmt(s)
	}
	sb.WriteString(p.sb.String())
	if e.Result != nil {
		for range g.indent + 1 {
			sb.WriteByte('\t')
		}
		fmt.Fprintf(&sb, "return %s\n", g.goExprStr(e.Result))
	}
	for range g.indent {
		sb.WriteByte('\t')
	}
	sb.WriteString("}()")
	return sb.String()
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
