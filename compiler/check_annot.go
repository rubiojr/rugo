package compiler

import (
	"fmt"
	"strings"

	"github.com/rubiojr/rugo/ast"
)

// annotCheck implements ast.Check and validates that every type annotation
// uses a recognised type name. Recognised names are returned by
// KnownTypeNames(); anything else is a user-facing compile error.
type annotCheck struct {
	sourceFile string
}

// TypeAnnotationCheck returns a Check that validates every "name : Type"
// annotation in def/fn parameters and return types. It does NOT enforce
// agreement between annotations and actual usage — that lives in the
// inferrer, where the type information is already gathered.
func TypeAnnotationCheck(sourceFile string) ast.Check {
	return &annotCheck{sourceFile: sourceFile}
}

func (a *annotCheck) Name() string { return "type-annotation" }

func (a *annotCheck) Check(prog *ast.Program) error {
	annotated := map[string]bool{}
	for _, s := range prog.Statements {
		if err := a.checkStmt(s, annotated); err != nil {
			return err
		}
	}
	return nil
}

func (a *annotCheck) checkStmt(s ast.Statement, annotated map[string]bool) error {
	switch st := s.(type) {
	case *ast.FuncDef:
		if err := validateFuncAnnotations(st, a.sourceFile); err != nil {
			return err
		}
		// Function bodies get their own annotation scope -- a `x : int = ...`
		// inside the body is independent of any outer binding.
		localAnnots := map[string]bool{}
		for _, child := range st.Body {
			if err := a.checkStmt(child, localAnnots); err != nil {
				return err
			}
		}
	case *ast.AssignStmt:
		if st.TypeAnnot != "" {
			if _, ok := ParseTypeAnnotation(st.TypeAnnot); !ok {
				return &ast.UserError{Msg: fmt.Sprintf(
					"%s:%d: unknown type %q in annotation for variable '%s' (valid types: %s)",
					a.sourceFile, st.SourceLine, st.TypeAnnot, st.Target, strings.Join(KnownTypeNames(), ", "),
				)}
			}
			if annotated[st.Target] {
				return &ast.UserError{Msg: fmt.Sprintf(
					"%s:%d: re-annotation of variable '%s' (annotations are sticky bindings — assign without `: T` to update an annotated variable)",
					a.sourceFile, st.SourceLine, st.Target,
				)}
			}
			annotated[st.Target] = true
		}
	case *ast.TestDef:
		localAnnots := map[string]bool{}
		for _, child := range st.Body {
			if err := a.checkStmt(child, localAnnots); err != nil {
				return err
			}
		}
	case *ast.BenchDef:
		localAnnots := map[string]bool{}
		for _, child := range st.Body {
			if err := a.checkStmt(child, localAnnots); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		for _, child := range st.Body {
			if err := a.checkStmt(child, annotated); err != nil {
				return err
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := a.checkStmt(child, annotated); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := a.checkStmt(child, annotated); err != nil {
				return err
			}
		}
	case *ast.WhileStmt:
		for _, child := range st.Body {
			if err := a.checkStmt(child, annotated); err != nil {
				return err
			}
		}
	case *ast.ForStmt:
		for _, child := range st.Body {
			if err := a.checkStmt(child, annotated); err != nil {
				return err
			}
		}
	}
	// Also walk any expressions in this statement for nested fn lambdas.
	var firstErr error
	walkStmtExprs(s, func(e ast.Expr) bool {
		if fn, ok := e.(*ast.FnExpr); ok {
			if err := a.checkFnExpr(fn); err != nil {
				firstErr = err
				return true
			}
		}
		return false
	})
	return firstErr
}

func (a *annotCheck) checkFnExpr(fn *ast.FnExpr) error {
	if err := validateFnExprAnnotations(fn, a.sourceFile); err != nil {
		return err
	}
	localAnnots := map[string]bool{}
	for _, child := range fn.Body {
		if err := a.checkStmt(child, localAnnots); err != nil {
			return err
		}
	}
	return nil
}

func validateFuncAnnotations(f *ast.FuncDef, sourceFile string) error {
	srcFile := f.SourceFile
	if srcFile == "" {
		srcFile = sourceFile
	}
	for _, p := range f.Params {
		if p.TypeAnnot == "" {
			continue
		}
		if _, ok := ParseTypeAnnotation(p.TypeAnnot); !ok {
			return &ast.UserError{Msg: fmt.Sprintf(
				"%s:%d: unknown type %q in annotation for parameter '%s' (valid types: %s)",
				srcFile, f.SourceLine, p.TypeAnnot, p.Name, strings.Join(KnownTypeNames(), ", "),
			)}
		}
	}
	if f.ReturnType != "" {
		if _, ok := ParseTypeAnnotation(f.ReturnType); !ok {
			return &ast.UserError{Msg: fmt.Sprintf(
				"%s:%d: unknown return type %q on function '%s' (valid types: %s)",
				srcFile, f.SourceLine, f.ReturnType, f.Name, strings.Join(KnownTypeNames(), ", "),
			)}
		}
	}
	return nil
}

func validateFnExprAnnotations(fn *ast.FnExpr, sourceFile string) error {
	for _, p := range fn.Params {
		if p.TypeAnnot == "" {
			continue
		}
		if _, ok := ParseTypeAnnotation(p.TypeAnnot); !ok {
			return &ast.UserError{Msg: fmt.Sprintf(
				"%s: unknown type %q in fn parameter '%s' (valid types: %s)",
				sourceFile, p.TypeAnnot, p.Name, strings.Join(KnownTypeNames(), ", "),
			)}
		}
	}
	if fn.ReturnType != "" {
		if _, ok := ParseTypeAnnotation(fn.ReturnType); !ok {
			return &ast.UserError{Msg: fmt.Sprintf(
				"%s: unknown return type %q on fn lambda (valid types: %s)",
				sourceFile, fn.ReturnType, strings.Join(KnownTypeNames(), ", "),
			)}
		}
	}
	return nil
}
