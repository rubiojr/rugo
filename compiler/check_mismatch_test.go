package compiler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompatibleWithAnnotation locks the mismatch-detection
// compatibility table for **return-context** checks: returning a value
// whose inferred type concretely conflicts with the annotated return
// type. Return-context is permissive because the codegen inserts
// numeric coercion (rugo_to_int, rugo_to_float) at the return site,
// and stringifies anything for a string-typed return.
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

		// `string` and `bool` annotations accept anything (runtime coerces).
		{"string annot, int inferred", TypeString, TypeInt, true},
		{"string annot, float inferred", TypeString, TypeFloat, true},
		{"string annot, bool inferred", TypeString, TypeBool, true},
		{"string annot, nil inferred", TypeString, TypeNil, true},
		{"bool annot, int inferred", TypeBool, TypeInt, true},
		{"bool annot, string inferred", TypeBool, TypeString, true},

		// Numeric family is mutually compatible.
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

// TestCheckMismatchAssignToAnnotatedParam exercises the full pipeline:
// reassigning an annotated `int` param to a string literal must produce
// a structured rugo-level error pointing at the assignment line.
func TestCheckMismatchAssignToAnnotatedParam(t *testing.T) {
	source := `
def f(a : int)
  a = "hello"
end
puts(f(1))
`
	err := compileSource(t, "mismatch.rugo", source)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "mismatch.rugo:3:")
	assert.Contains(t, msg, "cannot assign string value to parameter 'a' declared as int")
}

// TestCheckMismatchAssignToAnnotatedParamFloat verifies that
// reassigning an int-annotated parameter to a float literal is caught
// (assignment context is strict — generated Go has a concrete int
// variable, no coercion at the reassignment site).
func TestCheckMismatchAssignToAnnotatedParamFloat(t *testing.T) {
	source := `
def f(a : int)
  a = 3.14
end
puts(f(1))
`
	err := compileSource(t, "ok.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign float value to parameter 'a' declared as int")
}

// TestCheckMismatchAssignToAnnotatedStringParam verifies the strict
// rule for string annotations: reassigning to an int literal errors.
func TestCheckMismatchAssignToAnnotatedStringParam(t *testing.T) {
	source := `
def f(s : string)
  s = 42
end
f("hi")
`
	err := compileSource(t, "s.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign int value to parameter 's' declared as string")
}

// TestCheckMismatchAssignToAnnotatedAnyParam verifies that `any`
// remains permissive in assignment context.
func TestCheckMismatchAssignToAnnotatedAnyParam(t *testing.T) {
	source := `
def f(x : any)
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
def f() : int
  return "hi"
end
puts(f())
`
	err := compileSource(t, "ret.rugo", source)
	require.Error(t, err)
	msg := err.Error()
	assert.Contains(t, msg, "ret.rugo:3:")
	assert.Contains(t, msg, "cannot return string value from function declared returning int")
}

// TestCheckMismatchImplicitReturn locks return-type mismatch on the
// implicit-return (last-expression-as-value) form.
func TestCheckMismatchImplicitReturn(t *testing.T) {
	source := `
def f() : int
  "hi"
end
puts(f())
`
	err := compileSource(t, "impret.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot implicitly return string value")
}

// TestCheckMismatchInFnLambda verifies the check descends into fn lambdas.
func TestCheckMismatchInFnLambda(t *testing.T) {
	source := `
f = fn(a : int)
  a = "x"
end
f(1)
`
	err := compileSource(t, "lambda.rugo", source)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot assign string value to parameter 'a' declared as int")
}

// TestCheckMismatchSilentWhenDisableInfer makes sure --no-infer disables
// the check (no TypeInfo means no proof of conflict).
func TestCheckMismatchSilentWhenDisableInfer(t *testing.T) {
	source := `
def f() : int
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

// compileSource compiles a source string through the standard pipeline
// and returns the (possibly nil) error.
func compileSource(t *testing.T, sourceFile, src string) error {
	t.Helper()
	prog := parseProgram(t, src)
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
