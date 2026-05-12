package compiler

import (
	"strings"
	"testing"

	"github.com/rubiojr/rugo/ast"
	"github.com/stretchr/testify/assert"
)

// TestLiteralType covers the literalType helper which only recognises
// expressions whose type we can know without running the inferrer.
func TestLiteralType(t *testing.T) {
	cases := []struct {
		name   string
		expr   ast.Expr
		want   RugoType
		ok     bool
	}{
		{"Integer literal", &ast.IntLiteral{Value: "1"}, TypeInt, true},
		{"Float literal", &ast.FloatLiteral{Value: "1.5"}, TypeFloat, true},
		{"String literal", &ast.StringLiteral{Value: "x"}, TypeString, true},
		{"Bool literal", &ast.BoolLiteral{Value: true}, TypeBool, true},
		{"Nil literal", &ast.NilLiteral{}, TypeNil, true},
		{"Array literal", &ast.ArrayLiteral{}, TypeArray, true},
		{"Hash literal", &ast.HashLiteral{}, TypeHash, true},
		{
			"negative int via unary minus",
			&ast.UnaryExpr{Op: "-", Operand: &ast.IntLiteral{Value: "5"}},
			TypeInt, true,
		},
		{
			"negative float via unary minus",
			&ast.UnaryExpr{Op: "-", Operand: &ast.FloatLiteral{Value: "5.5"}},
			TypeFloat, true,
		},
		{
			"negated bool via unary !",
			&ast.UnaryExpr{Op: "!", Operand: &ast.BoolLiteral{Value: true}},
			TypeBool, true,
		},
		// Non-literals: variable refs, calls, dot expressions etc. — not flagged.
		{"identifier is not a literal", &ast.IdentExpr{Name: "x"}, TypeUnknown, false},
		{
			"call expr is not a literal",
			&ast.CallExpr{Func: &ast.IdentExpr{Name: "f"}},
			TypeUnknown, false,
		},
		{
			"binary expr is not a literal",
			&ast.BinaryExpr{Op: "+", Left: &ast.IntLiteral{Value: "1"}, Right: &ast.IntLiteral{Value: "2"}},
			TypeUnknown, false,
		},
		{
			"unary minus on non-literal is not a literal",
			&ast.UnaryExpr{Op: "-", Operand: &ast.IdentExpr{Name: "x"}},
			TypeUnknown, false,
		},
		{
			"unary ! on non-literal is not a literal",
			&ast.UnaryExpr{Op: "!", Operand: &ast.IdentExpr{Name: "x"}},
			TypeUnknown, false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := literalType(tc.expr)
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

// TestCheckCallSitesMismatch covers the call-site mismatch detector.
// Each row is a small source program; the expected substring must appear
// in the compile error message (or be empty if the program should compile
// cleanly).
func TestCheckCallSitesMismatch(t *testing.T) {
	cases := []struct {
		name        string
		source      string
		wantSubstr  string
		shouldError bool
	}{
		{
			name: "String literal to int param",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
puts(f("oops"))
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "Array literal to int param",
			source: `
def f(a : Integer)
  return a
end
puts(f([1, 2, 3]))
`,
			wantSubstr:  "cannot pass Array literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "Hash literal to array param",
			source: `
def f(xs : Array)
  return xs
end
puts(f({a: 1}))
`,
			wantSubstr:  "cannot pass Hash literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "Nil literal to int param",
			source: `
def f(a : Integer)
  return a
end
puts(f(nil))
`,
			wantSubstr:  "cannot pass Nil literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "Bool literal to array param",
			source: `
def f(xs : Array)
  return xs
end
puts(f(true))
`,
			wantSubstr:  "cannot pass Bool literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "second argument index reported correctly",
			source: `
def add(a : Integer, b : Integer) : Integer
  return a + b
end
puts(add(1, "x"))
`,
			wantSubstr:  "argument 2 to 'add'",
			shouldError: true,
		},
		// Permissive cases (no error)
		{
			name: "Integer literal to float param is permitted",
			source: `
def f(a : Float) : Float
  return a + 1.0
end
puts(f(2))
`,
			shouldError: false,
		},
		{
			name: "anything to any-annotated param is permitted",
			source: `
def f(x : Any) : Any
  return x
end
puts(f(nil))
puts(f("hi"))
puts(f([1, 2]))
`,
			shouldError: false,
		},
		{
			name: "negative integer literal to int param",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
puts(f(-5))
`,
			shouldError: false,
		},
		// Non-literal arguments (variable refs, calls) are not flagged.
		{
			name: "variable arg with dynamic type is not flagged",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
h = {1 => 2}
x = h["k"]
puts(f(x))
`,
			shouldError: false,
		},
		// Tier 3: callsite flow — variable arguments are flagged when
		// their flow-sensitive type is resolved to a concrete shape that
		// conflicts with the annotation. Dynamic / unresolved types still
		// pass silently (we cannot prove a mismatch).
		{
			name: "variable arg with concrete string is flagged (Tier 3)",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
x = "hello"
puts(f(x))
`,
			wantSubstr:  "cannot pass String value as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "variable arg sequential overwrite to incompat is flagged (Tier 3)",
			source: `
def f(a : Integer) : Integer
  return a
end
x = 1
x = "oops"
puts(f(x))
`,
			wantSubstr:  "cannot pass String value as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "variable arg sequential overwrite back to compat type passes (Tier 3)",
			source: `
def f(a : Integer) : Integer
  return a
end
x = "h"
x = 42
puts(f(x))
`,
			shouldError: false,
		},
		{
			name: "variable arg post-no-else-if is union and flagged (Tier 3)",
			source: `
def f(a : Integer) : Integer
  return a
end
x = 1
if true
  x = "oops"
end
puts(f(x))
`,
			wantSubstr:  "cannot pass",
			shouldError: true,
		},
		{
			name: "module dot-call is not flagged",
			source: `
use "str"
puts(str.upper(42))
`,
			shouldError: false,
		},
		{
			name: "unannotated function accepts Any literal",
			source: `
def f(a)
  return a
end
puts(f("anything"))
puts(f(42))
puts(f([1, 2]))
`,
			shouldError: false,
		},
		// Function with defaults still has its annotations honoured.
		{
			name: "function with defaults still validates literals",
			source: `
def add(a : Integer = 1, b : Integer = 2) : Integer
  return a + b
end
puts(add("oops", 10))
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'add'",
			shouldError: true,
		},
		{
			name: "function with defaults accepts valid literals",
			source: `
def add(a : Integer = 1, b : Integer = 2) : Integer
  return a + b
end
puts(add(5, 10))
`,
			shouldError: false,
		},
		// Nested contexts
		{
			name: "mismatch inside fn lambda body is caught",
			source: `
def square(n : Integer) : Integer
  return n * n
end
f = fn()
  return square("oops")
end
puts(f())
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'square'",
			shouldError: true,
		},
		{
			name: "mismatch in recursive self-call is caught",
			source: `
def f(n : Integer) : Integer
  if n <= 0
    return 0
  end
  return f("oops")
end
puts(f(3))
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "mismatch inside if condition is caught",
			source: `
def is_pos(n : Integer) : Bool
  return n > 0
end
if is_pos("oops")
  puts("yes")
end
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'is_pos'",
			shouldError: true,
		},
		{
			name: "mismatch inside try expression is caught",
			source: `
def f(n : Integer) : Integer
  return n * 2
end
result = try f("oops") or 99
puts(result)
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "mismatch inside spawn body is caught",
			source: `
def square(n : Integer) : Integer
  return n * n
end
task = spawn
  square("nope")
end
puts(task.value)
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'square'",
			shouldError: true,
		},
		// ---------------------------------------------------------------
		// Strict call-site rule: String / Bool / Nil params no longer
		// accept arbitrary types (matches the strict variable-annotation
		// rule). Numeric carve-out preserved (Int↔Float, Bool→numeric).
		// ---------------------------------------------------------------
		{
			name: "Integer literal to string param is flagged (strict)",
			source: `
def f(a : String) : String
  return a
end
puts(f(42))
`,
			wantSubstr:  "cannot pass Integer literal as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "Float literal to string param is flagged (strict)",
			source: `
def f(a : String) : String
  return a
end
puts(f(3.14))
`,
			wantSubstr:  "cannot pass Float literal as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "Bool literal to string param is flagged (strict)",
			source: `
def f(a : String) : String
  return a
end
puts(f(true))
`,
			wantSubstr:  "cannot pass Bool literal as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "Nil literal to string param is flagged (strict)",
			source: `
def f(a : String) : String
  return a
end
puts(f(nil))
`,
			wantSubstr:  "cannot pass Nil literal as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "Array literal to string param is flagged (strict)",
			source: `
def f(a : String) : String
  return a
end
puts(f([1, 2]))
`,
			wantSubstr:  "cannot pass Array literal as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "String literal to bool param is flagged (strict)",
			source: `
def f(a : Bool) : Bool
  return a
end
puts(f("hi"))
`,
			wantSubstr:  "cannot pass String literal as argument 1 to 'f' (parameter 'a' declared as Bool)",
			shouldError: true,
		},
		{
			name: "Integer literal to bool param is flagged (strict)",
			source: `
def f(a : Bool) : Bool
  return a
end
puts(f(0))
`,
			wantSubstr:  "cannot pass Integer literal as argument 1 to 'f' (parameter 'a' declared as Bool)",
			shouldError: true,
		},
		{
			name: "Array literal to bool param is flagged (strict)",
			source: `
def f(a : Bool) : Bool
  return a
end
puts(f([1, 2]))
`,
			wantSubstr:  "cannot pass Array literal as argument 1 to 'f' (parameter 'a' declared as Bool)",
			shouldError: true,
		},
		{
			name: "Tier 3: typed Integer variable to string param is flagged",
			source: `
def f(a : String) : String
  return a
end
x : Integer = 42
puts(f(x))
`,
			wantSubstr:  "cannot pass Integer value as argument 1 to 'f' (parameter 'a' declared as String)",
			shouldError: true,
		},
		{
			name: "Tier 3: typed String variable to bool param is flagged",
			source: `
def f(a : Bool) : Bool
  return a
end
x : String = "hi"
puts(f(x))
`,
			wantSubstr:  "cannot pass String value as argument 1 to 'f' (parameter 'a' declared as Bool)",
			shouldError: true,
		},
		// ---------------------------------------------------------------
		// Asymmetric numeric carve-out: Integer widens to Float and
		// Bool flows into Integer (0/1). Float → Integer is rejected
		// (lossy), Bool → Float is rejected (runtime helper would panic).
		// ---------------------------------------------------------------
		{
			name: "Integer literal to float param is permitted (widening)",
			source: `
def f(a : Float) : Float
  return a + 1.0
end
puts(f(2))
`,
			shouldError: false,
		},
		{
			name: "Float literal to int param is rejected (lossy truncation)",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
puts(f(2.5))
`,
			shouldError: true,
		},
		{
			name: "Bool literal to int param is permitted (0/1 semantics)",
			source: `
def f(a : Integer) : Integer
  return a + 1
end
puts(f(true))
`,
			shouldError: false,
		},
		{
			name: "Bool literal to float param is rejected (would panic at runtime)",
			source: `
def f(a : Float) : Float
  return a + 1.0
end
puts(f(true))
`,
			shouldError: true,
		},
		// ---------------------------------------------------------------
		// Any annot still accepts anything.
		// ---------------------------------------------------------------
		{
			name: "anything to any-annotated param is still permitted",
			source: `
def f(x : Any) : Any
  return x
end
puts(f(nil))
puts(f("hi"))
puts(f([1, 2]))
`,
			shouldError: false,
		},
		{
			name: "Same-type matches still pass: String to String",
			source: `
def f(s : String) : String
  return s
end
puts(f("hello"))
`,
			shouldError: false,
		},
		{
			name: "Same-type matches still pass: Bool to Bool",
			source: `
def f(b : Bool) : Bool
  return !b
end
puts(f(true))
`,
			shouldError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := compileSource(t, "call.rugo", tc.source)
			if tc.shouldError {
				if assert.Error(t, err) && tc.wantSubstr != "" {
					assert.Contains(t, err.Error(), tc.wantSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got: %s", err.Error())
			}
		})
	}
}

// TestCheckCallSitesErrorPointsAtSourceLine verifies the error includes the rugo line.
func TestCheckCallSitesErrorPointsAtSourceLine(t *testing.T) {
	source := `def f(a : Integer) : Integer
  return a + 1
end

puts(f("oops"))
`
	err := compileSource(t, "call.rugo", source)
	assert.Error(t, err)
	if err != nil {
		// The bad call is on line 5.
		assert.True(t, strings.Contains(err.Error(), ":5:"),
			"expected line :5: in error, got: %s", err.Error())
		assert.Contains(t, err.Error(), "call.rugo")
	}
}

// TestCheckCallSitesNoInferDisabled — the call-site check is skipped when --no-infer.
func TestCheckCallSitesNoInferDisabled(t *testing.T) {
	source := `
def f(a : Integer) : Integer
  return a + 1
end
puts(f("oops"))
`
	// generate(..., disableInfer=true) skips the mismatch checks entirely.
	_, err := generate(parseProgram(t, source), "call.rugo", false, nil, false, true)
	assert.NoError(t, err, "--no-infer should skip the call-site mismatch check")
}
