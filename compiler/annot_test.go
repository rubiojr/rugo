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
def add(a : Integer, b : Integer) : Integer
  return a + b
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "add")
	require.NotNil(t, f, "function 'add' not found")

	require.Len(t, f.Params, 2)
	assert.Equal(t, "Integer", f.Params[0].TypeAnnot)
	assert.Equal(t, "Integer", f.Params[1].TypeAnnot)
	assert.Equal(t, "Integer", f.ReturnType)
}

func TestParseAnnotationsMixed(t *testing.T) {
	src := `
def f(x, y : String, z) : Bool
  return x == y
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "f")
	require.NotNil(t, f)

	assert.Equal(t, "", f.Params[0].TypeAnnot, "unannotated param keeps empty TypeAnnot")
	assert.Equal(t, "String", f.Params[1].TypeAnnot)
	assert.Equal(t, "", f.Params[2].TypeAnnot)
	assert.Equal(t, "Bool", f.ReturnType)
}

func TestParseAnnotationsWithDefault(t *testing.T) {
	src := `
def greet(name : String = "world") : String
  return name
end
`
	prog := parseProgram(t, src)
	f := findFunc(prog, "greet")
	require.NotNil(t, f)

	assert.Equal(t, "String", f.Params[0].TypeAnnot)
	assert.NotNil(t, f.Params[0].Default, "default value preserved alongside annotation")
	assert.Equal(t, "String", f.ReturnType)
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
	// v0.29-era lowercase forms are rejected too (the canonical
	// vocabulary is now capitalised).
	_, ok = ParseTypeAnnotation("int")
	assert.False(t, ok, "lowercase 'int' is no longer a recognised annotation")
	_, ok = ParseTypeAnnotation("string")
	assert.False(t, ok, "lowercase 'string' is no longer a recognised annotation")
}

func TestParseTypeAnnotationMapping(t *testing.T) {
	cases := map[string]RugoType{
		"Integer": TypeInt,
		"Float":   TypeFloat,
		"String":  TypeString,
		"Bool":    TypeBool,
		"Array":   TypeArray,
		"Hash":    TypeHash,
		"Nil":     TypeNil,
		"Any":     TypeDynamic,
	}
	for name, want := range cases {
		got, ok := ParseTypeAnnotation(name)
		require.True(t, ok)
		assert.Equal(t, want, got, "ParseTypeAnnotation(%q)", name)
	}
}

func TestLegacyLowercaseAnnotationMapping(t *testing.T) {
	cases := map[string]string{
		"int":    "Integer",
		"float":  "Float",
		"string": "String",
		"bool":   "Bool",
		"array":  "Array",
		"hash":   "Hash",
		"nil":    "Nil",
		"any":    "Any",
	}
	for legacy, canonical := range cases {
		got, ok := LegacyLowercaseAnnotation(legacy)
		require.True(t, ok, "%q should be recognised as legacy lowercase", legacy)
		assert.Equal(t, canonical, got)
	}
	_, ok := LegacyLowercaseAnnotation("Integer")
	assert.False(t, ok, "already-canonical names are not legacy lowercase")
	_, ok = LegacyLowercaseAnnotation("integer")
	assert.False(t, ok, "misspellings are not legacy lowercase")
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
	assert.Contains(t, err.Error(), "Integer, Float, String, Bool")
}

func TestLegacyLowercaseAnnotationGivesHint(t *testing.T) {
	tmp := t.TempDir() + "/legacy.rugo"
	require.NoError(t, writeFile(tmp, `def f(x : int) : int
  return x
end
puts(f(1))`))
	c := &Compiler{}
	_, err := c.Compile(tmp)
	require.Error(t, err, "lowercase 'int' must be rejected as a v0.29-era spelling")
	msg := err.Error()
	assert.Contains(t, msg, "unknown type \"int\"")
	assert.Contains(t, msg, "did you mean \"Integer\"")
}

func TestUnknownReturnTypeIsCompileError(t *testing.T) {
	tmp := t.TempDir() + "/bad.rugo"
	require.NoError(t, writeFile(tmp, `def f(x) : integer return x end
puts(f(1))`))
	c := &Compiler{}
	_, err := c.Compile(tmp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown type")
	assert.Contains(t, err.Error(), "integer")
}

func TestInferRespectsParamAnnotation(t *testing.T) {
	src := `
def add(a : Integer, b : Integer) : Integer
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
def length(s : String) : Integer
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
double = fn(x : Integer) : Integer
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
	assert.Equal(t, "Integer", found.Params[0].TypeAnnot)
	assert.Equal(t, "Integer", found.ReturnType)
}

func TestStatsTracksAnnotated(t *testing.T) {
	src := `
def annotated(a : Integer, b : Integer) : Integer
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
