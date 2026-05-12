package compiler

import (
	"fmt"

	"github.com/rubiojr/rugo/ast"
)

// checkMismatch runs annotation/body mismatch detection.
//
// Two classes of conflict are flagged:
//
// Body-level — inside any annotated def or fn lambda, flag operations
// that the inferrer can prove disagree with the annotation:
//
//  1. Reassigning an annotated parameter to a value whose inferred type
//     concretely conflicts with the parameter's annotation.
//
//  2. Returning an expression whose inferred type concretely conflicts
//     with the function's annotated return type. This covers explicit
//     `return` statements and last-expression-as-return implicit returns.
//
// Call-site — at every direct call site to a user-defined function with
// annotated parameters, flag literal arguments whose type concretely
// conflicts with the corresponding parameter's annotation. Variables and
// computed expressions are never flagged (inference is conservative).
//
// The check never fires when either side is dynamic/unknown — annotations
// are trusted assertions, so unresolved inference must stay silent.
//
// Two compatibility predicates apply: a permissive one at return and
// call sites (codegen inserts numeric coercion / stringifies) and a
// strict one at body-level reassignment sites (the parameter has a
// concrete Go type with no coercion at the reassignment).
func checkMismatch(prog *ast.Program, ti *TypeInfo, sourceFile string) error {
	if ti == nil {
		return nil
	}
	// Body-level conflicts inside annotated def/fn bodies.
	for _, s := range prog.Statements {
		if err := checkMismatchStmt(s, ti, sourceFile, nil, ""); err != nil {
			return err
		}
	}
	// Call-site argument conflicts against annotated callees.
	return checkCallSites(prog, sourceFile)
}

// checkMismatchStmt walks a statement looking for annotated def/fn bodies
// to validate. paramAnnots and retAnnot describe the *enclosing* function
// (nil/zero when we're at the top level or inside an unannotated context).
func checkMismatchStmt(s ast.Statement, ti *TypeInfo, sourceFile string, paramAnnots map[string]RugoType, retAnnot string) error {
	switch st := s.(type) {
	case *ast.FuncDef:
		return checkMismatchFunc(st, ti, sourceFile)

	case *ast.AssignStmt:
		if paramAnnots != nil {
			if pa, ok := paramAnnots[st.Target]; ok {
				if err := checkAssignValue(st, pa, ti, sourceFile); err != nil {
					return err
				}
			}
		}
		return checkMismatchExpr(st.Value, ti, sourceFile)

	case *ast.ReturnStmt:
		if retAnnot != "" && st.Value != nil {
			if err := checkReturnValue(st.Value, st.SourceLine, retAnnot, ti, sourceFile, false); err != nil {
				return err
			}
		}
		if st.Value != nil {
			return checkMismatchExpr(st.Value, ti, sourceFile)
		}
		return nil

	case *ast.ImplicitReturnStmt:
		if retAnnot != "" {
			if err := checkReturnValue(st.Value, st.SourceLine, retAnnot, ti, sourceFile, true); err != nil {
				return err
			}
		}
		return checkMismatchExpr(st.Value, ti, sourceFile)

	case *ast.ExprStmt:
		return checkMismatchExpr(st.Expression, ti, sourceFile)

	case *ast.IfStmt:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
				return err
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.CaseStmt:
		for _, oc := range st.OfClauses {
			for _, child := range oc.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.WhileStmt:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.ForStmt:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.TestDef:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, nil, ""); err != nil {
				return err
			}
		}

	case *ast.BenchDef:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, nil, ""); err != nil {
				return err
			}
		}
	}
	return nil
}

// checkMismatchFunc validates the body of a function. Only checks fire on
// annotated params or annotated return; unannotated funcs are still walked
// so we descend into nested fn lambdas.
func checkMismatchFunc(f *ast.FuncDef, ti *TypeInfo, sourceFile string) error {
	annots := collectParamAnnots(f.Params)
	for _, child := range f.Body {
		if err := checkMismatchStmt(child, ti, fileFor(f, sourceFile), annots, f.ReturnType); err != nil {
			return err
		}
	}
	return nil
}

// checkMismatchExpr recurses into expressions to find nested fn lambdas
// whose bodies must also be validated.
func checkMismatchExpr(e ast.Expr, ti *TypeInfo, sourceFile string) error {
	var firstErr error
	walkExpr(e, func(node ast.Expr) bool {
		fn, ok := node.(*ast.FnExpr)
		if !ok {
			return false
		}
		annots := collectParamAnnots(fn.Params)
		for _, child := range fn.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, annots, fn.ReturnType); err != nil {
				firstErr = err
				return true
			}
		}
		return false
	})
	return firstErr
}

// checkAssignValue flags assignments that overwrite an annotated parameter
// with a value of a concretely conflicting type.
//
// Assignment context uses the **strict** compatibility rule: the
// generated Go declares the parameter with a concrete type (int, string,
// etc.), and there is no runtime coercion at the reassignment site. So
// `a = 3.14` to an int-annotated parameter is a clean rugo-level error
// rather than a confusing Go-level "cannot use float64 as int" message.
func checkAssignValue(st *ast.AssignStmt, paramAnnot RugoType, ti *TypeInfo, sourceFile string) error {
	inferred := ti.ExprType(st.Value)
	if compatibleAssignToAnnotation(paramAnnot, inferred) {
		return nil
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: cannot assign %s value to parameter '%s' declared as %s",
		sourceFile, st.SourceLine, displayTypeName(inferred), st.Target, displayTypeName(paramAnnot),
	)}
}

// checkReturnValue flags return statements whose value's inferred type
// concretely conflicts with the function's annotated return type.
func checkReturnValue(value ast.Expr, line int, retAnnot string, ti *TypeInfo, sourceFile string, implicit bool) error {
	annot, ok := ParseTypeAnnotation(retAnnot)
	if !ok {
		// Unknown type names are caught by TypeAnnotationCheck.
		return nil
	}
	inferred := ti.ExprType(value)
	if compatibleWithAnnotation(annot, inferred) {
		return nil
	}
	verb := "return"
	if implicit {
		verb = "implicitly return"
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: cannot %s %s value from function declared returning %s",
		sourceFile, line, verb, displayTypeName(inferred), displayTypeName(annot),
	)}
}

// displayTypeName returns the user-facing name for a RugoType. It differs
// from String() in two places: TypeFloat is shown as "float" (String()
// returns "float64" because it is used as a Go type name in codegen), and
// TypeDynamic is shown as "any" (the rugo source-level keyword).
func displayTypeName(t RugoType) string {
	switch t {
	case TypeFloat:
		return "float"
	case TypeDynamic:
		return "any"
	}
	return t.String()
}

// compatibleWithAnnotation reports whether a value with the given inferred
// type can flow into a **return** slot declared with the given annotation
// without the inferrer being able to prove a definite conflict.
//
// This is the *permissive* rule used at return sites: the codegen inserts
// numeric coercion wrappers (rugo_to_int, rugo_to_float) and stringifies
// anything for a string-typed return, so the numeric family is mutually
// compatible and `string`/`bool`/`any` annotations accept anything.
// Unresolved inference (`unknown`, `dynamic`) is always compatible.
//
// For assignment-context checks (reassigning an annotated parameter inside
// the body), use compatibleAssignToAnnotation, which is strict.
func compatibleWithAnnotation(annot, inferred RugoType) bool {
	if inferred == TypeUnknown || inferred == TypeDynamic {
		return true
	}
	switch annot {
	case TypeDynamic, TypeUnknown:
		return true
	case TypeString, TypeBool:
		return true
	case TypeInt, TypeFloat:
		return inferred == TypeInt || inferred == TypeFloat || inferred == TypeBool
	case TypeArray:
		return inferred == TypeArray
	case TypeHash:
		return inferred == TypeHash
	case TypeNil:
		return inferred == TypeNil
	}
	return true
}

// compatibleAssignToAnnotation is the **strict** counterpart used when
// reassigning an annotated parameter inside the function body. The
// generated Go declares the parameter with a concrete Go type, and there
// is no coercion at the reassignment site, so only same-type reassignment
// (plus `any`) is compatible. Unresolved inference (`unknown`, `dynamic`)
// is treated as compatible — there is no proof of conflict.
func compatibleAssignToAnnotation(annot, inferred RugoType) bool {
	if inferred == TypeUnknown || inferred == TypeDynamic {
		return true
	}
	switch annot {
	case TypeDynamic, TypeUnknown:
		return true
	}
	return annot == inferred
}

func collectParamAnnots(params []ast.Param) map[string]RugoType {
	var out map[string]RugoType
	for _, p := range params {
		if p.TypeAnnot == "" {
			continue
		}
		t, ok := ParseTypeAnnotation(p.TypeAnnot)
		if !ok {
			continue
		}
		if out == nil {
			out = make(map[string]RugoType, len(params))
		}
		out[p.Name] = t
	}
	return out
}

func fileFor(f *ast.FuncDef, fallback string) string {
	if f.SourceFile != "" {
		return f.SourceFile
	}
	return fallback
}
