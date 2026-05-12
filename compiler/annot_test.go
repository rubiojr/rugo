package compiler

import (
	"os"
	"testing"

	"github.com/rubiojr/rugo/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func parseProgram(t *testing.T, src string) *ast.Program {
	t.Helper()
	c := &Compiler{}
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	return prog
}

func findFunc(prog *ast.Program, name string) *ast.FuncDef {
	for _, s := range prog.Statements {
		if f, ok := s.(*ast.FuncDef); ok && f.Name == name {
			return f
		}
	}
	return nil
}

func TestParseAnnotationsParams(t *testing.T) {
	src := `
def add(a : int, b : int) : int
  return a + b
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "add")
	require.NotNil(t, f, "function 'add' not found")

	require.Len(t, f.Params, 2)
	assert.Equal(t, "int", f.Params[0].TypeAnnot)
	assert.Equal(t, "int", f.Params[1].TypeAnnot)
	assert.Equal(t, "int", f.ReturnType)
}

func TestParseAnnotationsMixed(t *testing.T) {
	src := `
def f(x, y : string, z) : bool
  return x == y
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "f")
	require.NotNil(t, f)

	assert.Equal(t, "", f.Params[0].TypeAnnot, "unannotated param keeps empty TypeAnnot")
	assert.Equal(t, "string", f.Params[1].TypeAnnot)
	assert.Equal(t, "", f.Params[2].TypeAnnot)
	assert.Equal(t, "bool", f.ReturnType)
}

func TestParseAnnotationsWithDefault(t *testing.T) {
	src := `
def greet(name : string = "world") : string
  return name
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "greet")
	require.NotNil(t, f)

	assert.Equal(t, "string", f.Params[0].TypeAnnot)
	assert.NotNil(t, f.Params[0].Default, "default value preserved alongside annotation")
	assert.Equal(t, "string", f.ReturnType)
}

func TestParseAnnotationsNoAnnotations(t *testing.T) {
	src := `
def plain(a, b)
  return a + b
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "plain")
	require.NotNil(t, f)

	for i, p := range f.Params {
		assert.Equal(t, "", p.TypeAnnot, "param %d should have empty TypeAnnot", i)
	}
	assert.Equal(t, "", f.ReturnType)
}

func TestParseTypeAnnotationKnownNames(t *testing.T) {
	for _, name := range KnownTypeNames() {
		_, ok := ParseTypeAnnotation(name)
		assert.True(t, ok, "%q should be a known type", name)
	}
}

func TestParseTypeAnnotationUnknown(t *testing.T) {
	_, ok := ParseTypeAnnotation("integer")
	assert.False(t, ok)
	_, ok = ParseTypeAnnotation("Int")
	assert.False(t, ok)
	_, ok = ParseTypeAnnotation("")
	assert.False(t, ok)
}

func TestParseTypeAnnotationMapping(t *testing.T) {
	cases := map[string]RugoType{
		"int":    TypeInt,
		"float":  TypeFloat,
		"string": TypeString,
		"bool":   TypeBool,
		"array":  TypeArray,
		"hash":   TypeHash,
		"nil":    TypeNil,
		"any":    TypeDynamic,
	}
	for name, want := range cases {
		got, ok := ParseTypeAnnotation(name)
		require.True(t, ok)
		assert.Equal(t, want, got, "ParseTypeAnnotation(%q)", name)
	}
}

func TestUnknownTypeIsCompileError(t *testing.T) {
	c := &Compiler{}
	_, err := c.ParseSource(`def f(x : integer) return x end`, "test.rugo")
	require.NoError(t, err, "parser accepts any ident; the check runs later")

	// Compile fully to trigger semantic check.
	tmp := t.TempDir() + "/bad.rugo"
	require.NoError(t, writeFile(tmp, `def f(x : integer) return x end
puts(f(1))`))
	_, err = c.Compile(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
	assert.Contains(t, err.Error(), "integer")
	assert.Contains(t, err.Error(), "int, float, string, bool")
}

func TestUnknownReturnTypeIsCompileError(t *testing.T) {
	tmp := t.TempDir() + "/bad.rugo"
	require.NoError(t, writeFile(tmp, `def f(x) : integer return x end
puts(f(1))`))
	c := &Compiler{}
	_, err := c.Compile(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown return type")
	assert.Contains(t, err.Error(), "integer")
}

func TestInferRespectsParamAnnotation(t *testing.T) {
	src := `
def add(a : int, b : int) : int
  return a + b
end
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	fti := ti.FuncTypes["add"]
	require.NotNil(t, fti)
	assert.Equal(t, TypeInt, fti.ParamTypes[0])
	assert.Equal(t, TypeInt, fti.ParamTypes[1])
	assert.Equal(t, TypeInt, fti.ReturnType)
	assert.Equal(t, []bool{true, true}, fti.AnnotatedArgs)
	assert.True(t, fti.AnnotatedReturn)
}

func TestAnnotationSeedsInferenceWhereInferenceWouldFail(t *testing.T) {
	// Without an annotation, a param only used via .method() would stay
	// dynamic. The annotation seeds it as int.
	src := `
def length(s : string) : int
  return len(s)
end
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	fti := ti.FuncTypes["length"]
	require.NotNil(t, fti)
	assert.Equal(t, TypeString, fti.ParamTypes[0])
	assert.Equal(t, TypeInt, fti.ReturnType)
}

func TestFnExprAnnotations(t *testing.T) {
	src := `
double = fn(x : int) : int
  return x * 2
end
puts(double(5))
`
	prog := parseProgram(t, src)
	require.NotNil(t, prog)

	var found *ast.FnExpr
	WalkExprs(prog, func(e ast.Expr) bool {
		if fn, ok := e.(*ast.FnExpr); ok {
			found = fn
			return true
		}
		return false
	})
	require.NotNil(t, found, "fn expression not found in AST")
	assert.Equal(t, "int", found.Params[0].TypeAnnot)
	assert.Equal(t, "int", found.ReturnType)
}

func TestStatsTracksAnnotated(t *testing.T) {
	src := `
def annotated(a : int, b : int) : int
  return a + b
end

def inferred(a, b)
  return a + b
end

puts(annotated(1, 2))
puts(inferred(3, 4))
`
	s := statsFor(t, src, false)
	assert.Equal(t, 4, s.Source.Params.Total)
	assert.Equal(t, 2, s.Source.Params.Annotated, "two annotated, two inferred")
	assert.Equal(t, 4, s.Source.Params.Typed, "both functions still infer int")

	assert.Equal(t, 2, s.Source.Returns.Total)
	assert.Equal(t, 1, s.Source.Returns.Annotated)
}

// writeFile is a tiny helper used by TestUnknown*Type tests above so we
// avoid pulling in os.WriteFile imports into the main test file body.
func writeFile(path, contents string) error {
	return os.WriteFile(path, []byte(contents), 0644)
}
