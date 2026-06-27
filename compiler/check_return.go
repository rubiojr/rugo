package compiler

import (
	"fmt"

	"github.com/rubiojr/rugo/ast"
)

// returnContextCheck implements ast.Check and reports `return` statements that
// appear outside of a function body — at the top level or inside top-level
// control flow. Without this check such code reaches `go build` and surfaces a
// raw Go error ("too many return values ... want ()").
//
// A plain *ast.ReturnStmt node only ever legitimately exists inside a def or fn
// body: returns inside spawn/parallel/try blocks are rewritten by the walker to
// dedicated node types (SpawnReturnStmt, TryHandlerReturnStmt), and lambda
// bodies live inside expressions. So the check walks top-level statements,
// descending only through the transparent control-flow blocks (if/while/for/
// case), and flags any ReturnStmt it reaches. It deliberately does NOT descend
// into function-like bodies (def/test/bench) or into expressions (fn/spawn/
// parallel/try), where a return is valid or already a different node type.
type returnContextCheck struct {
	sourceFile string
}

// ReturnContextCheck returns a Check that reports `return` used outside a function.
func ReturnContextCheck(sourceFile string) ast.Check {
	return &returnContextCheck{sourceFile: sourceFile}
}

func (rc *returnContextCheck) Name() string { return "return-context" }

func (rc *returnContextCheck) Check(prog *ast.Program) error {
	return rc.checkStmts(prog.Statements)
}

func (rc *returnContextCheck) checkStmts(stmts []ast.Statement) error {
	for _, s := range stmts {
		switch st := s.(type) {
		case *ast.ReturnStmt:
			return fmt.Errorf("%s:%d: return outside of a function (return is only valid inside a def or fn)", rc.sourceFile, st.StmtLine())
		case *ast.IfStmt:
			if err := rc.checkStmts(st.Body); err != nil {
				return err
			}
			for _, clause := range st.ElsifClauses {
				if err := rc.checkStmts(clause.Body); err != nil {
					return err
				}
			}
			if err := rc.checkStmts(st.ElseBody); err != nil {
				return err
			}
		case *ast.WhileStmt:
			if err := rc.checkStmts(st.Body); err != nil {
				return err
			}
		case *ast.ForStmt:
			if err := rc.checkStmts(st.Body); err != nil {
				return err
			}
		case *ast.CaseStmt:
			for _, oc := range st.OfClauses {
				if err := rc.checkStmts(oc.Body); err != nil {
					return err
				}
			}
			for _, clause := range st.ElsifClauses {
				if err := rc.checkStmts(clause.Body); err != nil {
					return err
				}
			}
			if err := rc.checkStmts(st.ElseBody); err != nil {
				return err
			}
		}
	}
	return nil
}
