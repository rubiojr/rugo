package compiler

import (
	"strings"
	"testing"

	"github.com/rubiojr/rugo/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompatibleWithAnnotation locks the mismatch-detection
// compatibility table for **return-context** checks: returning a value
// whose inferred type concretely conflicts with the annotated return
// type. The rule is strict (same as call-site) with a numeric
// runtime-coercion carve-out (rugo_to_int / rugo_to_float). Unlike
// earlier versions, `String`/`Bool` annotations do NOT silently accept
// arbitrary types — that hid real type errors.
func TestCompatibleWithAnnotation(t *testing.T) {
	cases := []struct {
		name       string
		annot      RugoType
		inferred   RugoType
		compatible bool
	}{
		// Unknown / dynamic on the inferred side always compatible.
		{"int annot, unknown inferred", TypeInt, TypeUnknown, true},
		{"int annot, dynamic inferred", TypeInt, TypeDynamic, true},
		{"string annot, dynamic inferred", TypeString, TypeDynamic, true},

		// `any` annotation accepts anything.
		{"any annot, int inferred", TypeDynamic, TypeInt, true},
		{"any annot, string inferred", TypeDynamic, TypeString, true},
		{"any annot, nil inferred", TypeDynamic, TypeNil, true},

		// Strict: `string` and `bool` annotations only accept matching values.
		{"string annot, string inferred", TypeString, TypeString, true},
		{"string annot, int inferred", TypeString, TypeInt, false},
		{"string annot, float inferred", TypeString, TypeFloat, false},
		{"string annot, bool inferred", TypeString, TypeBool, false},
		{"string annot, nil inferred", TypeString, TypeNil, false},
		{"string annot, array inferred", TypeString, TypeArray, false},
		{"string annot, hash inferred", TypeString, TypeHash, false},
		{"bool annot, bool inferred", TypeBool, TypeBool, true},
		{"bool annot, int inferred", TypeBool, TypeInt, false},
		{"bool annot, float inferred", TypeBool, TypeFloat, false},
		{"bool annot, string inferred", TypeBool, TypeString, false},
		{"bool annot, nil inferred", TypeBool, TypeNil, false},
		{"bool annot, array inferred", TypeBool, TypeArray, false},

		// Numeric family is mutually compatible (codegen coerces).
		{"int annot, int inferred", TypeInt, TypeInt, true},
		{"int annot, float inferred", TypeInt, TypeFloat, true},
		{"int annot, bool inferred", TypeInt, TypeBool, true},
		{"float annot, int inferred", TypeFloat, TypeInt, true},
		{"float annot, float inferred", TypeFloat, TypeFloat, true},
		{"float annot, bool inferred", TypeFloat, TypeBool, true},

		// Concrete conflicts that fire the check.
		{"int annot, string inferred", TypeInt, TypeString, false},
		{"int annot, array inferred", TypeInt, TypeArray, false},
		{"int annot, hash inferred", TypeInt, TypeHash, false},
		{"int annot, nil inferred", TypeInt, TypeNil, false},
		{"float annot, string inferred", TypeFloat, TypeString, false},
		{"float annot, nil inferred", TypeFloat, TypeNil, false},
		{"array annot, int inferred", TypeArray, TypeInt, false},
		{"array annot, hash inferred", TypeArray, TypeHash, false},
		{"array annot, string inferred", TypeArray, TypeString, false},
		{"hash annot, array inferred", TypeHash, TypeArray, false},
		{"hash annot, int inferred", TypeHash, TypeInt, false},
		{"nil annot, int inferred", TypeNil, TypeInt, false},
		{"nil annot, string inferred", TypeNil, TypeString, false},

		// Same-type matches.
		{"array annot, array inferred", TypeArray, TypeArray, true},
		{"hash annot, hash inferred", TypeHash, TypeHash, true},
		{"nil annot, nil inferred", TypeNil, TypeNil, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compatibleWithAnnotation(tc.annot, tc.inferred)
			assert.Equal(t, tc.compatible, got,
				"compatibleWithAnnotation(%s, %s)", tc.annot, tc.inferred)
		})
	}
}

// TestCompatibleAssignToAnnotation locks the compatibility table for
// **assignment-context** checks: reassigning an annotated parameter
// inside the function body. Assignment-context is strict because the
// generated Go gives the parameter a concrete Go type (int, float,
// string, bool) — there is no runtime coercion at the reassignment
// site, so a mismatched type would otherwise emit a confusing Go-level
// compile error.
func TestCompatibleAssignToAnnotation(t *testing.T) {
	cases := []struct {
		name       string
		annot      RugoType
		inferred   RugoType
		compatible bool
	}{
		// Unknown / dynamic on the inferred side: silent (no proof of conflict).
		{"int annot, unknown inferred", TypeInt, TypeUnknown, true},
		{"int annot, dynamic inferred", TypeInt, TypeDynamic, true},
		{"string annot, dynamic inferred", TypeString, TypeDynamic, true},

		// `any` annotation accepts anything.
		{"any annot, int inferred", TypeDynamic, TypeInt, true},
		{"any annot, string inferred", TypeDynamic, TypeString, true},
		{"any annot, nil inferred", TypeDynamic, TypeNil, true},

		// Same-type matches.
		{"int annot, int inferred", TypeInt, TypeInt, true},
		{"float annot, float inferred", TypeFloat, TypeFloat, true},
		{"string annot, string inferred", TypeString, TypeString, true},
		{"bool annot, bool inferred", TypeBool, TypeBool, true},
		{"array annot, array inferred", TypeArray, TypeArray, true},
		{"hash annot, hash inferred", TypeHash, TypeHash, true},
		{"nil annot, nil inferred", TypeNil, TypeNil, true},

		// Strict: numeric family is NOT mutually compatible in assignment.
		{"int annot, float inferred", TypeInt, TypeFloat, false},
		{"int annot, bool inferred", TypeInt, TypeBool, false},
		{"float annot, int inferred", TypeFloat, TypeInt, false},
		{"float annot, bool inferred", TypeFloat, TypeBool, false},

		// Strict: string/bool annotations do NOT accept everything.
		{"string annot, int inferred", TypeString, TypeInt, false},
		{"string annot, bool inferred", TypeString, TypeBool, false},
		{"string annot, nil inferred", TypeString, TypeNil, false},
		{"bool annot, int inferred", TypeBool, TypeInt, false},
		{"bool annot, string inferred", TypeBool, TypeString, false},

		// Concrete conflicts.
		{"int annot, string inferred", TypeInt, TypeString, false},
		{"int annot, array inferred", TypeInt, TypeArray, false},
		{"int annot, hash inferred", TypeInt, TypeHash, false},
		{"int annot, nil inferred", TypeInt, TypeNil, false},
		{"array annot, int inferred", TypeArray, TypeInt, false},
		{"array annot, hash inferred", TypeArray, TypeHash, false},
		{"hash annot, array inferred", TypeHash, TypeArray, false},
		{"nil annot, int inferred", TypeNil, TypeInt, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compatibleAssignToAnnotation(tc.annot, tc.inferred)
			assert.Equal(t, tc.compatible, got,
				"compatibleAssignToAnnotation(%s, %s)", tc.annot, tc.inferred)
		})
	}
}

// TestCompatibleCallArgToAnnotation locks the compatibility table for
// **call-site** checks: passing an argument whose inferred type
// concretely conflicts with the parameter's annotation. The call-site
// rule is strict (same as assignment) plus a numeric runtime-coercion
// carve-out: codegen inserts rugo_to_int / rugo_to_float wrappers at
// the call boundary, so Integer ↔ Float are mutually compatible, and
// Bool flows freely into numeric params (runtime treats it as 0/1).
//
// Unlike the assignment-context rule (compatibleAssignToAnnotation),
// the numeric carve-out also makes Integer and Float mutually
// compatible at call sites and at return sites. The return-context
// predicate (compatibleWithAnnotation) is now an alias of this one —
// `String` / `Bool` annotations no longer silently accept arbitrary
// values at returns either, matching the same strict rule a
// `x : String = ...` local variable enforces.
func TestCompatibleCallArgToAnnotation(t *testing.T) {
	cases := []struct {
		name       string
		annot      RugoType
		inferred   RugoType
		compatible bool
	}{
		// Unknown / dynamic on the inferred side: silent (no proof of conflict).
		{"int annot, unknown inferred", TypeInt, TypeUnknown, true},
		{"int annot, dynamic inferred", TypeInt, TypeDynamic, true},
		{"string annot, dynamic inferred", TypeString, TypeDynamic, true},

		// `any` annotation accepts anything.
		{"any annot, int inferred", TypeDynamic, TypeInt, true},
		{"any annot, string inferred", TypeDynamic, TypeString, true},
		{"any annot, nil inferred", TypeDynamic, TypeNil, true},
		{"any annot, array inferred", TypeDynamic, TypeArray, true},

		// Same-type matches.
		{"int annot, int inferred", TypeInt, TypeInt, true},
		{"float annot, float inferred", TypeFloat, TypeFloat, true},
		{"string annot, string inferred", TypeString, TypeString, true},
		{"bool annot, bool inferred", TypeBool, TypeBool, true},
		{"array annot, array inferred", TypeArray, TypeArray, true},
		{"hash annot, hash inferred", TypeHash, TypeHash, true},
		{"nil annot, nil inferred", TypeNil, TypeNil, true},

		// Numeric carve-out: Integer ↔ Float (codegen coerces).
		{"int annot, float inferred", TypeInt, TypeFloat, true},
		{"float annot, int inferred", TypeFloat, TypeInt, true},

		// Numeric carve-out: Bool flows into numeric (runtime 0/1).
		{"int annot, bool inferred", TypeInt, TypeBool, true},
		{"float annot, bool inferred", TypeFloat, TypeBool, true},

		// Strict: String / Bool no longer accept everything at call sites.
		{"string annot, int inferred", TypeString, TypeInt, false},
		{"string annot, float inferred", TypeString, TypeFloat, false},
		{"string annot, bool inferred", TypeString, TypeBool, false},
		{"string annot, nil inferred", TypeString, TypeNil, false},
		{"string annot, array inferred", TypeString, TypeArray, false},
		{"string annot, hash inferred", TypeString, TypeHash, false},
		{"bool annot, int inferred", TypeBool, TypeInt, false},
		{"bool annot, float inferred", TypeBool, TypeFloat, false},
		{"bool annot, string inferred", TypeBool, TypeString, false},
		{"bool annot, nil inferred", TypeBool, TypeNil, false},
		{"bool annot, array inferred", TypeBool, TypeArray, false},

		// Concrete conflicts (same as strict rule).
		{"int annot, string inferred", TypeInt, TypeString, false},
		{"int annot, array inferred", TypeInt, TypeArray, false},
		{"int annot, hash inferred", TypeInt, TypeHash, false},
		{"int annot, nil inferred", TypeInt, TypeNil, false},
		{"float annot, string inferred", TypeFloat, TypeString, false},
		{"float annot, nil inferred", TypeFloat, TypeNil, false},
		{"array annot, int inferred", TypeArray, TypeInt, false},
		{"array annot, hash inferred", TypeArray, TypeHash, false},
		{"array annot, string inferred", TypeArray, TypeString, false},
		{"hash annot, array inferred", TypeHash, TypeArray, false},
		{"hash annot, int inferred", TypeHash, TypeInt, false},
		{"nil annot, int inferred", TypeNil, TypeInt, false},
		{"nil annot, string inferred", TypeNil, TypeString, false},

		// Unions: every member must independently pass.
		{"int annot, Integer|Float union", TypeInt, TypeInt | TypeFloat, true},
		{"float annot, Integer|Float union", TypeFloat, TypeInt | TypeFloat, true},
		{"int annot, Integer|Bool union", TypeInt, TypeInt | TypeBool, true},
		{"int annot, Integer|String union", TypeInt, TypeInt | TypeString, false},
		{"string annot, String|Integer union", TypeString, TypeString | TypeInt, false},
		{"bool annot, Bool|Integer union", TypeBool, TypeBool | TypeInt, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := compatibleCallArgToAnnotation(tc.annot, tc.inferred)
			assert.Equal(t, tc.compatible, got,
				"compatibleCallArgToAnnotation(%s, %s)", tc.annot, tc.inferred)
		})
	}
}

// TestCheckMismatchAssignToAnnotatedParam exercises the full pipeline:
// reassigning an annotated `int` param to a String literal must produce
// a structured rugo-level error pointing at the assignment line.
func TestCheckMismatchAssignToAnnotatedParam(t *testing.T) {
	source := `
def f(a : Integer)
  a = "hello"
end
puts(f(1))
`
	err := compileSource(t, "mismatch.rugo", source)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "mismatch.rugo:3:")
	assert.Contains(t, msg, "cannot assign String value to parameter 'a' declared as Integer")
}

// TestCheckMismatchAssignToAnnotatedParamFloat verifies that
// reassigning an int-annotated parameter to a Float literal is caught
// (assignment context is strict — generated Go has a concrete int
// variable, no coercion at the reassignment site).
func TestCheckMismatchAssignToAnnotatedParamFloat(t *testing.T) {
	source := `
def f(a : Integer)
  a = 3.14
end
puts(f(1))
`
	err := compileSource(t, "ok.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign Float value to parameter 'a' declared as Integer")
}

// TestCheckMismatchAssignToAnnotatedStringParam verifies the strict
// rule for string annotations: reassigning to an Integer literal errors.
func TestCheckMismatchAssignToAnnotatedStringParam(t *testing.T) {
	source := `
def f(s : String)
  s = 42
end
f("hi")
`
	err := compileSource(t, "s.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign Integer value to parameter 's' declared as String")
}

// TestCheckMismatchAssignToAnnotatedAnyParam verifies that `any`
// remains permissive in assignment context.
func TestCheckMismatchAssignToAnnotatedAnyParam(t *testing.T) {
	source := `
def f(x : Any)
  x = "ok"
  x = 42
  x = [1, 2]
end
f(1)
`
	err := compileSource(t, "any.rugo", source)
	assert.NoError(t, err, "any annot should accept anything in assignment")
}

// TestCheckMismatchReturnTypeConflict locks return-type mismatch on an
// explicit `return` statement.
func TestCheckMismatchReturnTypeConflict(t *testing.T) {
	source := `
def f() : Integer
  return "hi"
end
puts(f())
`
	err := compileSource(t, "ret.rugo", source)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "ret.rugo:3:")
	assert.Contains(t, msg, "cannot return String value from function declared returning Integer")
}

// TestCheckMismatchImplicitReturn locks return-type mismatch on the
// implicit-return (last-expression-as-value) form.
func TestCheckMismatchImplicitReturn(t *testing.T) {
	source := `
def f() : Integer
  "hi"
end
puts(f())
`
	err := compileSource(t, "impret.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot implicitly return String value")
}

// TestCheckMismatchInFnLambda verifies the check descends into fn lambdas.
func TestCheckMismatchInFnLambda(t *testing.T) {
	source := `
f = fn(a : Integer)
  a = "x"
end
f(1)
`
	err := compileSource(t, "lambda.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign String value to parameter 'a' declared as Integer")
}

// TestCheckMismatchSilentWhenDisableInfer makes sure --no-infer disables
// the check (no TypeInfo means no proof of conflict).
func TestCheckMismatchSilentWhenDisableInfer(t *testing.T) {
	source := `
def f() : Integer
  return "hi"
end
puts(f())
`
	_, err := generate(parseProgram(t, source), "x.rugo", false, nil, false, true)
	assert.NoError(t, err, "--no-infer should skip the mismatch check")
}

// TestCheckMismatchUnannotated leaves unannotated functions alone.
func TestCheckMismatchUnannotated(t *testing.T) {
	source := `
def f(a)
  a = "x"
end
puts(f(1))
`
	err := compileSource(t, "ok.rugo", source)
	assert.NoError(t, err)
}

// TestTier3ReturnFlow_ReturnIdentMismatch verifies that returning a
// variable whose flow-sensitive type concretely conflicts with the
// declared return type is flagged, even though the value is not a literal.
func TestTier3ReturnFlow_ReturnIdentMismatch(t *testing.T) {
	source := `
def f() : Integer
  x = "hello"
  return x
end
puts(f())
`
	err := compileSource(t, "tier3ret.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot return String value from function declared returning Integer")
}

// TestTier3ReturnFlow_ReturnIdentNarrowedBack verifies that a variable
// reassigned back to a compatible type is permitted at the return site
// even though its storage union would include the earlier incompatible type.
func TestTier3ReturnFlow_ReturnIdentNarrowedBack(t *testing.T) {
	source := `
def f() : Integer
  x = "h"
  x = 42
  return x
end
puts(f())
`
	err := compileSource(t, "tier3retok.rugo", source)
	assert.NoError(t, err, "flow-sensitive narrowing should allow this return")
}

// TestTier3ReturnFlow_ImplicitReturnIdentMismatch covers the
// implicit-return (last-expression-as-value) form.
func TestTier3ReturnFlow_ImplicitReturnIdentMismatch(t *testing.T) {
	source := `
def f() : Integer
  x = "hello"
  x
end
puts(f())
`
	err := compileSource(t, "tier3impret.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot implicitly return String value")
}

// compileSource compiles a source string through the standard pipeline
// and returns the (possibly nil) error.
func compileSource(t *testing.T, sourceFile, src string) error {
	t.Helper()
	prog := parseProgram(t, src)
	// Run the same semantic checks the real Compile() pipeline runs so
	// tests cover TypeAnnotationCheck (unknown type names, re-annotation)
	// in addition to the generate-stage mismatch detection.
	checks := ast.CheckChain{
		TypeAnnotationCheck(sourceFile),
	}
	if err := checks.Run(prog); err != nil {
		return err
	}
	_, err := generate(prog, sourceFile, false, nil, false, false)
	if err != nil {
		// Sanity check that the error message references the rugo source,
		// not the generated Go. Surfaces regressions if the wrong location
		// info leaks into the message.
		assert.NotContains(t, err.Error(), ".go:", "error should not reference generated Go file")
		assert.Falsef(t, strings.HasPrefix(err.Error(), "cannot use"),
			"error should be a rugo-level diagnostic, got: %s", err.Error())
	}
	return err
}

// TestCompatibleWithAnnotationUnions exercises the union-aware
// extension of the return-context compatibility rule.
//
// Rule: a union is compatible with an annotation iff EVERY single-bit
// member of the union is independently compatible. This prevents a
// partially-resolved type (e.g. Integer|String) from sneaking through a
// narrow annotation by relying on the most-compatible member.
func TestCompatibleWithAnnotationUnions(t *testing.T) {
cases := []struct {
name       string
annot      RugoType
inferred   RugoType
compatible bool
}{
// Numeric-only union passes numeric annotation.
{"int annot, Integer|Float union", TypeInt, TypeInt | TypeFloat, true},
{"float annot, Integer|Float union", TypeFloat, TypeInt | TypeFloat, true},
// String/Bool annotations are strict: any non-matching member fails.
{"string annot, Integer|Nil union", TypeString, TypeInt | TypeNil, false},
{"string annot, Array|Hash union", TypeString, TypeArray | TypeHash, false},
{"bool annot, Integer|String union", TypeBool, TypeInt | TypeString, false},
// `any` annotation accepts any union.
{"any annot, String|Integer union", TypeDynamic, TypeString | TypeInt, true},
{"any annot, Hash|Nil union", TypeDynamic, TypeHash | TypeNil, true},
// Concrete conflicts: union contains a non-coercible member.
{"int annot, Integer|String union", TypeInt, TypeInt | TypeString, false},
{"int annot, Integer|Nil union", TypeInt, TypeInt | TypeNil, false},
{"int annot, Integer|Array union", TypeInt, TypeInt | TypeArray, false},
{"float annot, Float|String union", TypeFloat, TypeFloat | TypeString, false},
{"array annot, Array|Hash union", TypeArray, TypeArray | TypeHash, false},
{"hash annot, Hash|Array union", TypeHash, TypeHash | TypeArray, false},
{"nil annot, Nil|Integer union", TypeNil, TypeNil | TypeInt, false},
}
for _, tc := range cases {
t.Run(tc.name, func(t *testing.T) {
got := compatibleWithAnnotation(tc.annot, tc.inferred)
assert.Equal(t, tc.compatible, got,
"compatibleWithAnnotation(%s, %s)", tc.annot, tc.inferred)
})
}
}

// TestCompatibleAssignToAnnotationUnions exercises the union-aware
// extension of the strict assignment-context rule. Unions are strict:
// every member must match exactly.
func TestCompatibleAssignToAnnotationUnions(t *testing.T) {
cases := []struct {
name       string
annot      RugoType
inferred   RugoType
compatible bool
}{
// `any` accepts any union.
{"any annot, Integer|String union", TypeDynamic, TypeInt | TypeString, true},
{"any annot, Hash|Nil union", TypeDynamic, TypeHash | TypeNil, true},
// Strict: numeric union does NOT pass int (float member differs).
{"int annot, Integer|Float union", TypeInt, TypeInt | TypeFloat, false},
{"float annot, Integer|Float union", TypeFloat, TypeInt | TypeFloat, false},
// Strict: any cross-family union fails.
{"int annot, Integer|String union", TypeInt, TypeInt | TypeString, false},
{"string annot, Integer|String union", TypeString, TypeInt | TypeString, false},
{"array annot, Array|Hash union", TypeArray, TypeArray | TypeHash, false},
{"string annot, Integer|Nil union", TypeString, TypeInt | TypeNil, false},
{"bool annot, Integer|String union", TypeBool, TypeInt | TypeString, false},
}
for _, tc := range cases {
t.Run(tc.name, func(t *testing.T) {
got := compatibleAssignToAnnotation(tc.annot, tc.inferred)
assert.Equal(t, tc.compatible, got,
"compatibleAssignToAnnotation(%s, %s)", tc.annot, tc.inferred)
})
}
}

// TestVarAnnotInitMismatch asserts that the initial binding value must
// be compatible with the annotation: `x : Integer = "hi"` is a compile
// error.
func TestVarAnnotInitMismatch(t *testing.T) {
source := `
def main()
  x : Integer = "hello"
end
main()
`
err := compileSource(t, "init.rugo", source)
require.Error(t, err)
assert.Contains(t, err.Error(), "cannot assign String value to")
assert.Contains(t, err.Error(), "declared as Integer")
}

// TestVarAnnotReassignMismatch asserts that reassigning an annotated
// local with an incompatible type fires the strict assignment check.
func TestVarAnnotReassignMismatch(t *testing.T) {
source := `
def main()
  x : Integer = 0
  x = "hello"
end
main()
`
err := compileSource(t, "reassign.rugo", source)
require.Error(t, err)
assert.Contains(t, err.Error(), "cannot assign String value to")
}

// TestVarAnnotAnyIsPermissive asserts that `x : Any` silences the check
// for every subsequent assignment (the explicit suppression hatch).
func TestVarAnnotAnyIsPermissive(t *testing.T) {
source := `
def main()
  x : Any = 0
  x = "hello"
  x = [1, 2]
end
main()
`
err := compileSource(t, "any.rugo", source)
assert.NoError(t, err, "any annotation should accept any reassignment")
}

// TestVarAnnotUnknownType asserts that an unknown type name in an
// annotation is rejected with a clear error.
func TestVarAnnotUnknownType(t *testing.T) {
source := `
def main()
  x : foobar = 0
end
main()
`
err := compileSource(t, "unknown.rugo", source)
require.Error(t, err)
assert.Contains(t, err.Error(), "unknown type")
assert.Contains(t, err.Error(), "foobar")
}

// TestVarAnnotReannotationError asserts that re-annotating an already-
// annotated local in the same scope is a compile error.
func TestVarAnnotReannotationError(t *testing.T) {
source := `
def main()
  x : Integer = 0
  x : Integer = 1
end
main()
`
err := compileSource(t, "reannot.rugo", source)
require.Error(t, err)
assert.Contains(t, err.Error(), "re-annotation")
}

// TestVarAnnotUnannotatedUnchanged asserts that the unannotated path is
// unchanged — mixed reassignment is still permitted.
func TestVarAnnotUnannotatedUnchanged(t *testing.T) {
source := `
def main()
  x = 0
  x = "hello"
end
main()
`
err := compileSource(t, "unannot.rugo", source)
assert.NoError(t, err)
}

// TestVarAnnotNumericPromotion asserts that `x : Float = 0` allows
// reassignment with an Integer literal (numeric coercion).
func TestVarAnnotNumericPromotion(t *testing.T) {
source := `
def main()
  x : Float = 1.0
  x = 42
end
main()
`
// Strict assignment context does NOT permit cross-numeric reassignment
// (the Go variable is float64 — no implicit int→float conversion at the
// assignment site). The check_mismatch error makes this user-visible.
err := compileSource(t, "promote.rugo", source)
require.Error(t, err)
assert.Contains(t, err.Error(), "cannot assign Integer value to")
assert.Contains(t, err.Error(), "declared as Float")
}

// TestParamDefaultLiteralMismatch verifies that a default-value literal
// whose static type concretely conflicts with the parameter's annotation
// is rejected at compile time. The default expression is a contract:
// callers that omit the argument receive that exact value, so a literal
// mismatch is a guaranteed type-violation that the compiler can prove.
//
// The compatibility rule is the strict call-site rule (now also used
// at return sites): numeric types are mutually compatible, `Any`
// accepts anything, and every other annotation requires a same-type
// literal.
func TestParamDefaultLiteralMismatch(t *testing.T) {
	cases := []struct {
		name       string
		source     string
		wantSubstr string
	}{
		{
			name: "string default for int param",
			source: `
def f(x : Integer = "hi") : Integer
  return x
end
puts(f())
`,
			wantSubstr: "cannot use String literal as default for parameter 'x' declared as Integer",
		},
		{
			name: "nil default for int param",
			source: `
def f(x : Integer = nil) : Integer
  return x
end
puts(f())
`,
			wantSubstr: "cannot use Nil literal as default for parameter 'x' declared as Integer",
		},
		{
			name: "int default for array param",
			source: `
def f(x : Array = 42) : Integer
  return len(x)
end
puts(f())
`,
			wantSubstr: "cannot use Integer literal as default for parameter 'x' declared as Array",
		},
		{
			name: "array default for hash param",
			source: `
def f(x : Hash = [1, 2]) : Integer
  return len(x)
end
puts(f())
`,
			wantSubstr: "cannot use Array literal as default for parameter 'x' declared as Hash",
		},
		{
			name: "int default for nil param",
			source: `
def f(x : Nil = 42)
  puts(x)
end
f()
`,
			wantSubstr: "cannot use Integer literal as default for parameter 'x' declared as Nil",
		},
		{
			name: "fn lambda also checked",
			source: `
double = fn(n : Integer = "oops") : Integer
  return n
end
puts(double())
`,
			wantSubstr: "cannot use String literal as default for parameter 'n' declared as Integer",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := compileSource(t, "paramdef.rugo", tc.source)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSubstr)
		})
	}
}

// TestParamDefaultLiteralCompatible verifies the strict-with-numeric-
// carve-out rule for parameter defaults: the same rule the call-site
// checker uses (defaults flow into the param at call time).
//
// Numeric family (Integer/Float/Bool) flows freely between Integer and
// Float annotations; `Any` accepts anything; everything else must match
// the annotation exactly. String / Bool annotations no longer accept
// arbitrary defaults — that's covered by TestParamDefaultLiteralMismatch.
func TestParamDefaultLiteralCompatible(t *testing.T) {
	cases := []struct {
		name   string
		source string
	}{
		{"string param accepts string default", `
def f(x : String = "hi") : String
  return x
end
puts(f())
`},
		{"bool param accepts bool default", `
def f(x : Bool = true) : Bool
  return x
end
puts(f())
`},
		{"any param accepts anything", `
def f(x : Any = [1, 2, 3])
  puts(x)
end
f()
`},
		{"int param accepts float default (numeric carve-out)", `
def f(x : Integer = 1.5) : Integer
  return x + 1
end
puts(f())
`},
		{"float param accepts int default (numeric carve-out)", `
def f(x : Float = 1) : Float
  return x
end
puts(f())
`},
		{"int param accepts bool default (numeric carve-out)", `
def f(x : Integer = true) : Integer
  return x + 1
end
puts(f())
`},
		{"computed (non-literal) default is silently allowed", `
def helper() : Integer
  return 10
end
def f(x : Array = helper()) : Integer
  return 0
end
puts(f())
`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := compileSource(t, "paramdefok.rugo", tc.source)
			assert.NoError(t, err, "default should be accepted")
		})
	}
}

// TestParamDefaultLiteralStrictString verifies that under the strict
// call-site rule, a `: String` parameter rejects non-String literal
// defaults at compile time (an Integer default would silently flow
// into the param at every defaulted call site, which is the same kind
// of mismatch the call-site checker now flags for explicit args).
func TestParamDefaultLiteralStrictString(t *testing.T) {
	source := `
def f(x : String = 42) : String
  return x
end
puts(f())
`
	err := compileSource(t, "strictdef.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use Integer literal as default for parameter 'x' declared as String")
}

// TestParamDefaultLiteralStrictBool verifies the same for Bool defaults:
// an Integer literal is no longer accepted as a Bool default value.
func TestParamDefaultLiteralStrictBool(t *testing.T) {
	source := `
def f(x : Bool = "yes") : Bool
  return x
end
puts(f())
`
	err := compileSource(t, "strictdef.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot use String literal as default for parameter 'x' declared as Bool")
}
