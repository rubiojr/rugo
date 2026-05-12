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
				return &ast.UserError{Msg: unknownTypeMsg(
					a.sourceFile, st.SourceLine, st.TypeAnnot,
					fmt.Sprintf("annotation for variable '%s'", st.Target),
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
			return &ast.UserError{Msg: unknownTypeMsg(
				srcFile, f.SourceLine, p.TypeAnnot,
				fmt.Sprintf("annotation for parameter '%s'", p.Name),
			)}
		}
	}
	if f.ReturnType != "" {
		if _, ok := ParseTypeAnnotation(f.ReturnType); !ok {
			return &ast.UserError{Msg: unknownTypeMsg(
				srcFile, f.SourceLine, f.ReturnType,
				fmt.Sprintf("return type on function '%s'", f.Name),
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
			return &ast.UserError{Msg: unknownTypeMsgFile(
				sourceFile, p.TypeAnnot,
				fmt.Sprintf("fn parameter '%s'", p.Name),
			)}
		}
	}
	if fn.ReturnType != "" {
		if _, ok := ParseTypeAnnotation(fn.ReturnType); !ok {
			return &ast.UserError{Msg: unknownTypeMsgFile(
				sourceFile, fn.ReturnType, "return type on fn lambda",
			)}
		}
	}
	return nil
}

// unknownTypeMsg renders the "unknown type" diagnostic for a `file:line`
// source location. When the offending name is a v0.29-era lowercase form
// (e.g. "int"), the message includes a "did you mean Integer?" hint so
// users migrating from the early annotation syntax see exactly what to
// change.
func unknownTypeMsg(file string, line int, typeName, context string) string {
	if cap, ok := LegacyLowercaseAnnotation(typeName); ok {
		return fmt.Sprintf(
			"%s:%d: unknown type %q in %s — did you mean %q? (annotation type names are capitalised; valid types: %s)",
			file, line, typeName, context, cap, strings.Join(KnownTypeNames(), ", "),
		)
	}
	return fmt.Sprintf(
		"%s:%d: unknown type %q in %s (valid types: %s)",
		file, line, typeName, context, strings.Join(KnownTypeNames(), ", "),
	)
}

// unknownTypeMsgFile is the file-only variant used by fn lambdas, which
// historically lack a source-line column on the FnExpr node.
func unknownTypeMsgFile(file, typeName, context string) string {
	if cap, ok := LegacyLowercaseAnnotation(typeName); ok {
		return fmt.Sprintf(
			"%s: unknown type %q in %s — did you mean %q? (annotation type names are capitalised; valid types: %s)",
			file, typeName, context, cap, strings.Join(KnownTypeNames(), ", "),
		)
	}
	return fmt.Sprintf(
		"%s: unknown type %q in %s (valid types: %s)",
		file, typeName, context, strings.Join(KnownTypeNames(), ", "),
	)
}
