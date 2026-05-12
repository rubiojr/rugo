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
		{"int literal", &ast.IntLiteral{Value: "1"}, TypeInt, true},
		{"float literal", &ast.FloatLiteral{Value: "1.5"}, TypeFloat, true},
		{"string literal", &ast.StringLiteral{Value: "x"}, TypeString, true},
		{"bool literal", &ast.BoolLiteral{Value: true}, TypeBool, true},
		{"nil literal", &ast.NilLiteral{}, TypeNil, true},
		{"array literal", &ast.ArrayLiteral{}, TypeArray, true},
		{"hash literal", &ast.HashLiteral{}, TypeHash, true},
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
			name: "string literal to int param",
			source: `
def f(a : int) : int
  return a + 1
end
puts(f("oops"))
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "array literal to int param",
			source: `
def f(a : int)
  return a
end
puts(f([1, 2, 3]))
`,
			wantSubstr:  "cannot pass array literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "hash literal to array param",
			source: `
def f(xs : array)
  return xs
end
puts(f({a: 1}))
`,
			wantSubstr:  "cannot pass hash literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "nil literal to int param",
			source: `
def f(a : int)
  return a
end
puts(f(nil))
`,
			wantSubstr:  "cannot pass nil literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "bool literal to array param",
			source: `
def f(xs : array)
  return xs
end
puts(f(true))
`,
			wantSubstr:  "cannot pass bool literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "second argument index reported correctly",
			source: `
def add(a : int, b : int) : int
  return a + b
end
puts(add(1, "x"))
`,
			wantSubstr:  "argument 2 to 'add'",
			shouldError: true,
		},
		// Permissive cases (no error)
		{
			name: "int literal to float param is permitted",
			source: `
def f(a : float) : float
  return a + 1.0
end
puts(f(2))
`,
			shouldError: false,
		},
		{
			name: "int literal to string param is permitted",
			source: `
def f(a : string) : string
  return a
end
puts(f(42))
`,
			shouldError: false,
		},
		{
			name: "anything to any-annotated param is permitted",
			source: `
def f(x : any) : any
  return x
end
puts(f(nil))
puts(f("hi"))
puts(f([1, 2]))
`,
			shouldError: false,
		},
		{
			name: "anything to bool-annotated param is permitted",
			source: `
def f(a : bool) : bool
  return a
end
puts(f(0))
puts(f("hi"))
puts(f([1, 2]))
`,
			shouldError: false,
		},
		{
			name: "negative integer literal to int param",
			source: `
def f(a : int) : int
  return a + 1
end
puts(f(-5))
`,
			shouldError: false,
		},
		// Non-literal arguments (variable refs, calls) are not flagged.
		{
			name: "variable arg is not flagged",
			source: `
def f(a : int) : int
  return a + 1
end
x = "anything"
puts(f(x))
`,
			shouldError: false,
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
			name: "unannotated function accepts any literal",
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
def add(a : int = 1, b : int = 2) : int
  return a + b
end
puts(add("oops", 10))
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'add'",
			shouldError: true,
		},
		{
			name: "function with defaults accepts valid literals",
			source: `
def add(a : int = 1, b : int = 2) : int
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
def square(n : int) : int
  return n * n
end
f = fn()
  return square("oops")
end
puts(f())
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'square'",
			shouldError: true,
		},
		{
			name: "mismatch in recursive self-call is caught",
			source: `
def f(n : int) : int
  if n <= 0
    return 0
  end
  return f("oops")
end
puts(f(3))
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "mismatch inside if condition is caught",
			source: `
def is_pos(n : int) : bool
  return n > 0
end
if is_pos("oops")
  puts("yes")
end
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'is_pos'",
			shouldError: true,
		},
		{
			name: "mismatch inside try expression is caught",
			source: `
def f(n : int) : int
  return n * 2
end
result = try f("oops") or 99
puts(result)
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'f'",
			shouldError: true,
		},
		{
			name: "mismatch inside spawn body is caught",
			source: `
def square(n : int) : int
  return n * n
end
task = spawn
  square("nope")
end
puts(task.value)
`,
			wantSubstr:  "cannot pass string literal as argument 1 to 'square'",
			shouldError: true,
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
	source := `def f(a : int) : int
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
def f(a : int) : int
  return a + 1
end
puts(f("oops"))
`
	// generate(..., disableInfer=true) skips the mismatch checks entirely.
	_, err := generate(parseProgram(t, source), "call.rugo", false, nil, false, true)
	assert.NoError(t, err, "--no-infer should skip the call-site mismatch check")
}
