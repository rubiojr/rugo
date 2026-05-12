package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findFirstAssign returns the first AssignStmt at the top level of the
// program. Used by the variable-annotation tests below to inspect the
// walker's output.
func findFirstAssign(t *testing.T, prog *Program) *AssignStmt {
	t.Helper()
	for _, st := range prog.Statements {
		if a, ok := st.(*AssignStmt); ok {
			return a
		}
	}
	t.Fatalf("no AssignStmt found in program")
	return nil
}

func parseVarAnnotSource(t *testing.T, src string) (*Program, error) {
	t.Helper()
	c := &Compiler{}
	return c.ParseSource(src, "test.rugo")
}

// TestVarAnnotParsesTypeAnnot exercises the `x : T = expr` syntax for
// each of the 8 recognised type names. Each test asserts the resulting
// AssignStmt carries the expected TypeAnnot string.
func TestVarAnnotParsesTypeAnnot(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{"int", "x : int = 42", "int"},
		{"float", "x : float = 3.14", "float"},
		{"string", `x : string = "hi"`, "string"},
		{"bool", "x : bool = true", "bool"},
		{"array", "x : array = [1, 2, 3]", "array"},
		{"hash", "x : hash = {1 => 2}", "hash"},
		{"nil", "x : nil = nil", "nil"},
		{"any", "x : any = 42", "any"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			prog, err := parseVarAnnotSource(t, tc.src)
			require.NoError(t, err)
			a := findFirstAssign(t, prog)
			assert.Equal(t, tc.want, a.TypeAnnot,
				"AssignStmt.TypeAnnot for %q", tc.src)
		})
	}
}

// TestVarAnnotEmptyWithoutAnnotation asserts that `x = 42` without an
// annotation produces AssignStmt.TypeAnnot == "".
func TestVarAnnotEmptyWithoutAnnotation(t *testing.T) {
	prog, err := parseVarAnnotSource(t, "x = 42")
	require.NoError(t, err)
	a := findFirstAssign(t, prog)
	assert.Equal(t, "", a.TypeAnnot)
}

// TestVarAnnotRejectsIndexAndDotAssign asserts that index and field
// assignments cannot carry a type annotation (those are mutations, not
// bindings).
func TestVarAnnotRejectsIndexAndDotAssign(t *testing.T) {
	cases := []struct {
		name string
		src  string
	}{
		{"index assign", "arr = [1, 2]\narr[0] : int = 99"},
		{"dot assign", "obj.field : int = 5"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseVarAnnotSource(t, tc.src)
			assert.Error(t, err, "expected parse/walk error for %q", tc.src)
		})
	}
}
