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
	// Default-value annotation conflicts (literal-only). Runs before the
	// flow-sensitive checks because default expressions are part of the
	// function signature itself, so a violation should be reported as
	// soon as the signature is encountered.
	if err := checkParamDefaults(prog, sourceFile); err != nil {
		return err
	}
	// Top-level annotated `x : T = ...` bindings live in the program scope.
	topAnnots := collectLocalAnnots(prog.Statements, nil)
	// Body-level conflicts inside annotated def/fn bodies AND top-level
	// reassignment of annotated top-level locals.
	for _, s := range prog.Statements {
		if err := checkMismatchStmt(s, ti, sourceFile, nil, topAnnots, ""); err != nil {
			return err
		}
	}
	// Call-site argument conflicts against annotated callees.
	return checkCallSites(prog, ti, sourceFile)
}

// checkMismatchStmt walks a statement looking for annotated def/fn bodies
// to validate. paramAnnots describes the *enclosing* function's params
// (nil at the top level or inside an unannotated context). localAnnots
// describes annotated local bindings (`x : T = ...`) in the current
// function/test/bench scope.
func checkMismatchStmt(s ast.Statement, ti *TypeInfo, sourceFile string, paramAnnots, localAnnots map[string]RugoType, retAnnot string) error {
	switch st := s.(type) {
	case *ast.FuncDef:
		return checkMismatchFunc(st, ti, sourceFile)

	case *ast.AssignStmt:
		if paramAnnots != nil {
			if pa, ok := paramAnnots[st.Target]; ok {
				if err := checkAssignValue(st, pa, "parameter", ti, sourceFile); err != nil {
					return err
				}
			}
		}
		if localAnnots != nil {
			if la, ok := localAnnots[st.Target]; ok {
				if err := checkAssignValue(st, la, "variable", ti, sourceFile); err != nil {
					return err
				}
			}
		}
		return checkMismatchExpr(st.Value, ti, sourceFile)

	case *ast.ReturnStmt:
		if retAnnot != "" {
			if st.Value == nil {
				if err := checkBareReturn(st.SourceLine, retAnnot, sourceFile); err != nil {
					return err
				}
			} else if err := checkReturnValue(st.Value, st.SourceLine, retAnnot, ti, sourceFile, false); err != nil {
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
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
				return err
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.CaseStmt:
		for _, oc := range st.OfClauses {
			for _, child := range oc.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.WhileStmt:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.ForStmt:
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, retAnnot); err != nil {
				return err
			}
		}

	case *ast.TestDef:
		annots := collectLocalAnnots(st.Body, nil)
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, nil, annots, ""); err != nil {
				return err
			}
		}

	case *ast.BenchDef:
		annots := collectLocalAnnots(st.Body, nil)
		for _, child := range st.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, nil, annots, ""); err != nil {
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
	paramAnnots := collectParamAnnots(f.Params)
	localAnnots := collectLocalAnnots(f.Body, nil)
	for _, child := range f.Body {
		if err := checkMismatchStmt(child, ti, fileFor(f, sourceFile), paramAnnots, localAnnots, f.ReturnType); err != nil {
			return err
		}
	}
	if err := checkFallOff(f.ReturnType, f.Body, f.Name, f.SourceLine, fileFor(f, sourceFile)); err != nil {
		return err
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
		paramAnnots := collectParamAnnots(fn.Params)
		localAnnots := collectLocalAnnots(fn.Body, nil)
		for _, child := range fn.Body {
			if err := checkMismatchStmt(child, ti, sourceFile, paramAnnots, localAnnots, fn.ReturnType); err != nil {
				firstErr = err
				return true
			}
		}
		if err := checkFallOff(fn.ReturnType, fn.Body, "<lambda>", fnExprLine(fn), sourceFile); err != nil {
			firstErr = err
			return true
		}
		return false
	})
	return firstErr
}

// fnExprLine returns a representative source line for an FnExpr node.
// FnExpr itself has no SourceLine field, so we use the first body
// statement's line, falling back to 0 (which results in a "file:0:"
// prefix the user can ignore).
func fnExprLine(fn *ast.FnExpr) int {
	if len(fn.Body) == 0 {
		return 0
	}
	return fn.Body[0].StmtLine()
}

// checkAssignValue flags assignments that overwrite an annotated
// parameter or local variable with a value of a concretely conflicting
// type.
//
// Assignment context uses the **strict** compatibility rule: the
// generated Go declares the binding with a concrete type (int, string,
// etc.), and there is no runtime coercion at the reassignment site. So
// `a = 3.14` to an int-annotated binding is a clean rugo-level error
// rather than a confusing Go-level "cannot use float64 as int" message.
//
// kind is the user-facing noun ("parameter" or "variable") used in the
// error message.
func checkAssignValue(st *ast.AssignStmt, annot RugoType, kind string, ti *TypeInfo, sourceFile string) error {
	// Lambda literals (`fn(...) ... end`) infer as TypeDynamic, which the
	// compatibility predicate treats as "no proof of conflict". Detect
	// them syntactically so they cannot silently flow into a typed slot.
	if _, isFn := st.Value.(*ast.FnExpr); isFn && annot != TypeDynamic && annot != TypeUnknown {
		return &ast.UserError{Msg: fmt.Sprintf(
			"%s:%d: cannot assign Lambda value to %s '%s' declared as %s",
			sourceFile, st.SourceLine, kind, st.Target, displayTypeName(annot),
		)}
	}
	inferred := ti.ExprType(st.Value)
	if compatibleAssignToAnnotation(annot, inferred) {
		return nil
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: cannot assign %s value to %s '%s' declared as %s",
		sourceFile, st.SourceLine, displayTypeName(inferred), kind, st.Target, displayTypeName(annot),
	)}
}

// checkBareReturn flags a bare `return` (no value) inside a function
// whose return annotation cannot accept Nil. A bare return is
// conceptually `return nil`, so any annotation other than `Nil` or
// `Any` is a concrete type-violation the compiler can prove. Without
// this check the violation would either fall through to a Go-level
// diagnostic that leaks lowercase Go type names (`int`, `string`,
// `float64`, `bool`) or be silently accepted for reference-typed
// annotations (`Array`, `Hash`) where Go's nil happens to be a valid
// zero value.
func checkBareReturn(line int, retAnnot, sourceFile string) error {
	annot, ok := ParseTypeAnnotation(retAnnot)
	if !ok {
		// Unknown type names are caught by TypeAnnotationCheck.
		return nil
	}
	if compatibleWithAnnotation(annot, TypeNil) {
		return nil
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: cannot return without a value from function declared returning %s",
		sourceFile, line, displayTypeName(annot),
	)}
}

// checkFallOff flags functions whose body can reach the closing `end`
// without producing a return value. When the return annotation is `Nil`
// or `Any` (or absent), falling off the end is fine — the caller's
// contract permits a nil result. For any other annotation, every path
// through the body must terminate in a `return`, an implicit-return
// tail expression, or an exhaustive if/case where every branch itself
// always returns. Without this check the failure mode is either a
// silent nil result (for unused/reference-typed slots) or a runtime
// panic from the generated rugo_to_X coercion call.
//
// The check runs AFTER ImplicitReturnLowering: tail expressions in
// function bodies and tail branches of if/case statements have already
// been wrapped as ImplicitReturnStmt, so the analyzer only needs to
// look for explicit/implicit return nodes and exhaustive branches.
// Reuses bodyAlwaysReturns from codegen_stmt.go for the path analysis.
func checkFallOff(retAnnot string, body []ast.Statement, name string, line int, sourceFile string) error {
	if retAnnot == "" {
		return nil
	}
	annot, ok := ParseTypeAnnotation(retAnnot)
	if !ok {
		// Unknown type names are caught by TypeAnnotationCheck.
		return nil
	}
	// `Nil` and `Any` return slots accept missing-return; everything
	// else demands an unconditional return on every path.
	if compatibleWithAnnotation(annot, TypeNil) {
		return nil
	}
	if bodyAlwaysReturns(body) {
		return nil
	}
	if name == "" || name == "<lambda>" {
		return &ast.UserError{Msg: fmt.Sprintf(
			"%s:%d: missing return at end of function declared returning %s",
			sourceFile, line, displayTypeName(annot),
		)}
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: missing return at end of function '%s' declared returning %s",
		sourceFile, line, name, displayTypeName(annot),
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
	inferred := returnFlowType(value, ti)
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

// returnFlowType returns the flow-sensitive type of a return-statement
// value: for IdentExpr nodes that's VarUseTypes (the type the variable
// holds at the return site, replace semantics across reassignments), and
// for other expression shapes it falls back to ExprTypes.
//
// Tier 3 return-flow: returning a variable that was narrowed back to a
// compatible type via a later reassignment is permitted, even though the
// storage union for that variable includes earlier (now-overwritten)
// incompatible types.
func returnFlowType(e ast.Expr, ti *TypeInfo) RugoType {
	if ti == nil {
		return TypeDynamic
	}
	if _, isIdent := e.(*ast.IdentExpr); isIdent {
		if t, ok := ti.VarUseTypes[e]; ok {
			return t
		}
	}
	return ti.ExprType(e)
}

// displayTypeName returns the user-facing name for a RugoType. It differs
// from String() in one place: TypeFloat is shown as "Float" (String()
// returns "float64" only via GoType() — note: String() returns "Float").
// In practice this is now a thin wrapper around String() that exists for
// historical symmetry with code paths that pre-date the unified
// annotation vocabulary.
func displayTypeName(t RugoType) string {
	switch t {
	case TypeFloat:
		return "Float"
	case TypeDynamic:
		return "Any"
	}
	return t.String()
}

// compatibleWithAnnotation reports whether a value with the given inferred
// type can flow into a **return** slot declared with the given annotation
// without the inferrer being able to prove a definite conflict.
//
// The rule is strict (same as call-site and assignment-context), with a
// numeric runtime-coercion carve-out: codegen inserts rugo_to_int /
// rugo_to_float wrappers at the return boundary, so `Integer` and
// `Float` are mutually compatible and `Bool` flows freely into either
// numeric return type. `String`/`Bool` annotations do NOT silently
// accept other types — that surprised users and hid real type errors.
// `Any` annotations and `Unknown`/`Dynamic` inferred types remain
// silent (no proof of conflict).
//
// In practice this is the same predicate as compatibleCallArgToAnnotation;
// they are kept as separate names because the two code paths invoke them
// in semantically different contexts.
//
// For assignment-context checks (reassigning an annotated parameter inside
// the body), use compatibleAssignToAnnotation, which is strict with no
// numeric carve-out (the parameter has a concrete Go type and there is
// no coercion at the reassignment site).
func compatibleWithAnnotation(annot, inferred RugoType) bool {
	return compatibleCallArgToAnnotation(annot, inferred)
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
	if inferred.IsUnion() {
		for _, m := range inferred.Members() {
			if !compatibleAssignToAnnotation(annot, m) {
				return false
			}
		}
		return true
	}
	return annot == inferred
}

// compatibleCallArgToAnnotation is the predicate used at call sites:
// validating an argument's inferred type against the callee's parameter
// annotation. It is strict like compatibleAssignToAnnotation (so a
// `: String` param does NOT silently accept an Integer literal, matching
// the same rule a `x : String = ...` local variable enforces), with a
// narrow asymmetric numeric carve-out:
//
//   - Integer → Float widening at the call boundary (codegen inserts
//     rugo_to_float). Float → Integer is rejected because it would
//     silently truncate (e.g. `f(1.5)` against `: Integer`).
//   - Bool → Integer is accepted (runtime treats bool as 0/1; codegen
//     inserts rugo_to_int). Bool → Float is rejected because the
//     runtime helper does not handle that conversion and would panic.
//
// `Any` annotations and `Unknown`/`Dynamic` inferred types remain
// silent. Unions are checked member-wise: every member must
// independently pass, so an Integer|String value can't sneak into a
// `: Integer` parameter. An Integer|Float union no longer fits a
// `: Integer` parameter (Float member fails), but Integer|Bool does
// (both members pass).
//
// Use this for call-site checks (literal args, variable args) and for
// parameter default-value checks. The return-context predicate
// (compatibleWithAnnotation) is a thin alias of this function — return
// slots use the same strict rule with the asymmetric numeric carve-out.
func compatibleCallArgToAnnotation(annot, inferred RugoType) bool {
	if inferred == TypeUnknown || inferred == TypeDynamic {
		return true
	}
	switch annot {
	case TypeDynamic, TypeUnknown:
		return true
	}
	if inferred.IsUnion() {
		for _, m := range inferred.Members() {
			if !compatibleCallArgToAnnotation(annot, m) {
				return false
			}
		}
		return true
	}
	if annot == inferred {
		return true
	}
	// Asymmetric numeric carve-out: only widening / well-defined
	// conversions are silent. Codegen inserts rugo_to_float /
	// rugo_to_int wrappers at the call boundary for these cases.
	if annot == TypeFloat && inferred == TypeInt {
		return true
	}
	if annot == TypeInt && inferred == TypeBool {
		return true
	}
	return false
}

// collectLocalAnnots scans a statement list (a function body, top-level
// program, or test/bench block) for `x : T = expr` bindings and adds
// them to dst. First-appearance wins (sticky semantics; re-annotation
// is rejected by TypeAnnotationCheck before we get here).
//
// The scan descends into if/case/while/for/try bodies because Rugo has
// function-level scoping — an annotation introduced inside a branch is
// visible across the entire function.
//
// It does NOT descend into nested FuncDef / FnExpr / TestDef / BenchDef
// bodies; each of those owns its own scope.
func collectLocalAnnots(stmts []ast.Statement, dst map[string]RugoType) map[string]RugoType {
	for _, s := range stmts {
		switch st := s.(type) {
		case *ast.AssignStmt:
			if st.TypeAnnot == "" {
				continue
			}
			if _, already := dst[st.Target]; already {
				continue
			}
			t, ok := ParseTypeAnnotation(st.TypeAnnot)
			if !ok {
				continue
			}
			if dst == nil {
				dst = make(map[string]RugoType)
			}
			dst[st.Target] = t
		case *ast.IfStmt:
			dst = collectLocalAnnots(st.Body, dst)
			for _, c := range st.ElsifClauses {
				dst = collectLocalAnnots(c.Body, dst)
			}
			dst = collectLocalAnnots(st.ElseBody, dst)
		case *ast.CaseStmt:
			for _, oc := range st.OfClauses {
				dst = collectLocalAnnots(oc.Body, dst)
			}
			for _, c := range st.ElsifClauses {
				dst = collectLocalAnnots(c.Body, dst)
			}
			dst = collectLocalAnnots(st.ElseBody, dst)
		case *ast.WhileStmt:
			dst = collectLocalAnnots(st.Body, dst)
		case *ast.ForStmt:
			dst = collectLocalAnnots(st.Body, dst)
		}
	}
	return dst
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

// checkParamDefaults flags parameter default values whose static literal
// type concretely conflicts with the parameter's annotation. The default
// expression is part of the signature contract (callers omitting the
// argument receive that exact value), so a literal mismatch is a
// guaranteed type-violation that the compiler can prove without any
// inference.
//
// Only literal defaults (numbers, strings, bools, nil, array/hash
// literals, and unary-`-`/`!` of those) are checked. Computed defaults
// (function calls, identifiers) are silently allowed because their value
// could legitimately match the annotation.
//
// The compatibility rule is the strict-with-numeric-carve-out one
// (compatibleCallArgToAnnotation), matching call-site checks:
// `String`/`Bool` annotations only accept their own type, the numeric
// family (`Integer`/`Float`/`Bool`) flows freely between numeric
// annotations, and `Any` accepts anything. A `: String` default value
// of `42` is therefore a guaranteed type-violation.
//
// Both top-level FuncDefs and FnExpr lambdas (anywhere they appear) are
// validated. Nested FuncDefs are reachable through walkStmtExprs's
// statement-body recursion.
func checkParamDefaults(prog *ast.Program, sourceFile string) error {
	for _, s := range prog.Statements {
		if err := checkParamDefaultsStmt(s, sourceFile); err != nil {
			return err
		}
	}
	return nil
}

func checkParamDefaultsStmt(s ast.Statement, sourceFile string) error {
	switch st := s.(type) {
	case *ast.FuncDef:
		file := fileFor(st, sourceFile)
		for _, p := range st.Params {
			if err := checkParamDefault(p, file, st.SourceLine); err != nil {
				return err
			}
		}
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, file); err != nil {
				return err
			}
		}
	case *ast.TestDef:
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
	case *ast.BenchDef:
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
	case *ast.IfStmt:
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
		for _, c := range st.ElsifClauses {
			for _, child := range c.Body {
				if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
					return err
				}
			}
		}
		for _, child := range st.ElseBody {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
	case *ast.WhileStmt:
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
	case *ast.ForStmt:
		for _, child := range st.Body {
			if err := checkParamDefaultsStmt(child, sourceFile); err != nil {
				return err
			}
		}
	}
	// Walk expressions in this statement for nested FnExpr lambdas.
	var firstErr error
	walkStmtExprs(s, func(e ast.Expr) bool {
		fn, ok := e.(*ast.FnExpr)
		if !ok {
			return false
		}
		line := stmtLine(s)
		for _, p := range fn.Params {
			if err := checkParamDefault(p, sourceFile, line); err != nil {
				firstErr = err
				return true
			}
		}
		return false
	})
	return firstErr
}

// checkParamDefault validates a single Param's default expression against
// its annotation. Returns nil when there is no annotation, no default,
// the annotation is unknown (caught elsewhere), or the default has no
// statically-resolvable type (identifiers, function calls, etc.).
//
// Both true literals (`42`, `-3.14`) and compound expressions whose
// type can be folded from their operands (`1.0 + 0.5`, `"a" + "b"`,
// `1 < 2`) are checked. The error message distinguishes literals
// ("X literal") from compound expressions ("X value") so the user can
// tell which form of default they wrote.
func checkParamDefault(p ast.Param, sourceFile string, line int) error {
	if p.TypeAnnot == "" || p.Default == nil {
		return nil
	}
	annot, ok := ParseTypeAnnotation(p.TypeAnnot)
	if !ok {
		return nil
	}
	if litType, isLit := literalType(p.Default); isLit {
		if compatibleCallArgToAnnotation(annot, litType) {
			return nil
		}
		return &ast.UserError{Msg: fmt.Sprintf(
			"%s:%d: cannot use %s literal as default for parameter '%s' declared as %s",
			sourceFile, line, displayTypeName(litType), p.Name, displayTypeName(annot),
		)}
	}
	exprType, isStatic := staticExprType(p.Default)
	if !isStatic {
		return nil
	}
	if compatibleCallArgToAnnotation(annot, exprType) {
		return nil
	}
	return &ast.UserError{Msg: fmt.Sprintf(
		"%s:%d: cannot use %s value as default for parameter '%s' declared as %s",
		sourceFile, line, displayTypeName(exprType), p.Name, displayTypeName(annot),
	)}
}

// stmtLine returns the source line of a statement when available, or 0
// when the statement type does not embed BaseStmt.
func stmtLine(s ast.Statement) int {
	type lineGetter interface{ StmtLine() int }
	if g, ok := s.(lineGetter); ok {
		return g.StmtLine()
	}
	return 0
}
