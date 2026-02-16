package compiler

import (
	"github.com/rubiojr/rugo/ast"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSource(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseSource(`puts("hello")`, "test.rugo")
	require.NoError(t, err)
	assert.Equal(t, "test.rugo", prog.SourceFile)
	assert.Len(t, prog.Statements, 1)
}

func TestParseSourceFuncDef(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseSource("def greet(name)\n  puts(name)\nend\n", "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Statements, 1)
	fn, ok := prog.Statements[0].(*ast.FuncDef)
	require.True(t, ok)
	assert.Equal(t, "greet", fn.Name)
	assert.Equal(t, []ast.Param{{Name: "name"}}, fn.Params)
	assert.Len(t, fn.Body, 1)
}

func TestParseSourceError(t *testing.T) {
	c := &Compiler{}
	_, err := c.ParseSource("if\nend\n", "bad.rugo")
	assert.Error(t, err)
}

func TestParseFile(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseFile("../examples/hello.rugo")
	require.NoError(t, err)
	assert.NotEmpty(t, prog.Statements)
	assert.Equal(t, "../examples/hello.rugo", prog.SourceFile)
}

func TestEndLineFuncDef(t *testing.T) {
	c := &Compiler{}
	src := "def foo()\n  x = 1\n  y = 2\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Statements, 1)
	fn := prog.Statements[0].(*ast.FuncDef)
	assert.Equal(t, 1, fn.StmtLine())
	assert.Equal(t, 4, fn.StmtEndLine())
}

func TestEndLineTestDef(t *testing.T) {
	c := &Compiler{TestMode: true}
	src := "use \"test\"\nrats \"my test\"\n  assert(true)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.True(t, len(prog.Statements) >= 2)
	td, ok := prog.Statements[1].(*ast.TestDef)
	require.True(t, ok, "expected ast.TestDef, got %T", prog.Statements[1])
	assert.Equal(t, 2, td.StmtLine())
	assert.Equal(t, 4, td.StmtEndLine())
}

func TestEndLineIfStmt(t *testing.T) {
	c := &Compiler{}
	src := "if true\n  puts(1)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Statements, 1)
	ifStmt := prog.Statements[0].(*ast.IfStmt)
	assert.Equal(t, 1, ifStmt.StmtLine())
	assert.Equal(t, 3, ifStmt.StmtEndLine())
}

func TestEndLineNonBlockStmt(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseSource("x = 42\n", "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Statements, 1)
	assign := prog.Statements[0].(*ast.AssignStmt)
	assert.Equal(t, 1, assign.StmtLine())
	assert.Equal(t, 1, assign.StmtEndLine())
}

func TestEndLineMultipleFuncs(t *testing.T) {
	c := &Compiler{}
	src := "def foo()\n  puts(1)\nend\n\ndef bar()\n  puts(2)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Statements, 2)
	fn1 := prog.Statements[0].(*ast.FuncDef)
	fn2 := prog.Statements[1].(*ast.FuncDef)
	assert.Equal(t, 1, fn1.StmtLine())
	assert.Equal(t, 3, fn1.StmtEndLine())
	assert.Equal(t, 5, fn2.StmtLine())
	assert.Equal(t, 7, fn2.StmtEndLine())
}

func TestStmtLineAfterHeredoc(t *testing.T) {
	c := &Compiler{}
	// Heredoc compresses 3 lines (opener, body, delimiter) into 1.
	// def foo() should still report original line 6, not a shifted line.
	src := "msg = <<TEXT\nHello\nTEXT\n\ndef foo()\n  puts(msg)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	lines := strings.Split(prog.RawSource, "\n")
	var fn *ast.FuncDef
	for _, s := range prog.Statements {
		if f, ok := s.(*ast.FuncDef); ok {
			fn = f
			break
		}
	}
	require.NotNil(t, fn)
	assert.Equal(t, 5, fn.StmtLine())
	assert.Equal(t, 7, fn.StmtEndLine())
	// The raw source line at StmtLine() should be the actual def
	assert.Equal(t, "def foo()", strings.TrimSpace(lines[fn.StmtLine()-1]))
}

func TestInferExported(t *testing.T) {
	c := &Compiler{}
	src := "def add(a, b)\n  return a + b\nend\nx = add(1, 2)\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	ti := Infer(prog)
	assert.NotNil(t, ti)
	assert.NotEmpty(t, ti.FuncTypes)
	fti, ok := ti.FuncTypes["add"]
	require.True(t, ok)
	assert.Equal(t, TypeInt, fti.ReturnType)
}

func TestWalkStmts(t *testing.T) {
	c := &Compiler{}
	src := "def foo()\n  x = 1\nend\ny = 2\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	var names []string
	WalkStmts(prog, func(s ast.Statement) bool {
		switch st := s.(type) {
		case *ast.FuncDef:
			names = append(names, "def:"+st.Name)
		case *ast.AssignStmt:
			names = append(names, "assign:"+st.Target)
		}
		return true
	})
	assert.Equal(t, []string{"def:foo", "assign:x", "assign:y"}, names)
}

func TestWalkStmtsSkipChildren(t *testing.T) {
	c := &Compiler{}
	src := "def foo()\n  x = 1\nend\ny = 2\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	var names []string
	WalkStmts(prog, func(s ast.Statement) bool {
		switch st := s.(type) {
		case *ast.FuncDef:
			names = append(names, "def:"+st.Name)
			return false // skip body
		case *ast.AssignStmt:
			names = append(names, "assign:"+st.Target)
		}
		return true
	})
	// x=1 is inside foo's body, should be skipped
	assert.Equal(t, []string{"def:foo", "assign:y"}, names)
}

func TestWalkExprsExported(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseSource("x = 1 + 2\n", "test.rugo")
	require.NoError(t, err)

	var foundBinary bool
	WalkExprs(prog, func(e ast.Expr) bool {
		if _, ok := e.(*ast.BinaryExpr); ok {
			foundBinary = true
			return true
		}
		return false
	})
	assert.True(t, foundBinary)
}

func TestRawSourcePreserved(t *testing.T) {
	c := &Compiler{}
	src := "# greet the user\ndef greet(name)\n  puts(name)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	assert.Equal(t, src, prog.RawSource)
	assert.Contains(t, prog.RawSource, "# greet the user")
}

func TestRawSourceFromFile(t *testing.T) {
	c := &Compiler{}
	prog, err := c.ParseFile("../examples/hello.rugo")
	require.NoError(t, err)
	assert.NotEmpty(t, prog.RawSource)
	assert.Contains(t, prog.RawSource, "puts")
}

func TestRawSourceCommentCorrelation(t *testing.T) {
	c := &Compiler{}
	src := "# adds two numbers\ndef add(a, b)\n  return a + b\nend\n\n# no doc here\nx = 1\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	lines := strings.Split(prog.RawSource, "\n")

	// ast.FuncDef at line 2 should have comment at line 1
	fn := prog.Statements[0].(*ast.FuncDef)
	assert.Equal(t, 2, fn.StmtLine())
	commentLine := lines[fn.StmtLine()-2] // line above the def
	assert.Equal(t, "# adds two numbers", commentLine)

	// ast.AssignStmt at line 7 has a comment at line 6 but with a blank line gap
	assign := prog.Statements[1].(*ast.AssignStmt)
	assert.Equal(t, 7, assign.StmtLine())
	gapLine := lines[assign.StmtLine()-2] // line above is "# no doc here"
	assert.Equal(t, "# no doc here", gapLine)
}

func TestRawSourceMultiLineDocComment(t *testing.T) {
	c := &Compiler{}
	src := "# first line of doc\n# second line of doc\ndef foo()\n  puts(1)\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	lines := strings.Split(prog.RawSource, "\n")
	fn := prog.Statements[0].(*ast.FuncDef)
	assert.Equal(t, 3, fn.StmtLine())

	// Walk backwards from the def to collect doc comment lines
	var docLines []string
	for i := fn.StmtLine() - 2; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "#") {
			docLines = append([]string{trimmed}, docLines...)
		} else {
			break
		}
	}
	assert.Equal(t, []string{"# first line of doc", "# second line of doc"}, docLines)
}

func TestRawSourceInlineCommentStripped(t *testing.T) {
	c := &Compiler{}
	src := "x = 42 # the answer\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	// AST has the assignment without the comment
	assign := prog.Statements[0].(*ast.AssignStmt)
	assert.Equal(t, "x", assign.Target)

	// But RawSource still has the inline comment
	assert.Contains(t, prog.RawSource, "# the answer")
}

func TestRawSourceExtractBlockWithComments(t *testing.T) {
	c := &Compiler{}
	src := "# helper function\ndef greet(name)\n  # say hello\n  puts(name)\nend\n\ndef main()\n  greet(\"world\")\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)

	lines := strings.Split(prog.RawSource, "\n")

	// Extract the greet function block using position info
	fn := prog.Statements[0].(*ast.FuncDef)
	assert.Equal(t, "greet", fn.Name)

	// Include the doc comment above
	startLine := fn.StmtLine() - 1 // 0-indexed
	for startLine > 0 && strings.HasPrefix(strings.TrimSpace(lines[startLine-1]), "#") {
		startLine--
	}
	endLine := fn.StmtEndLine() // 1-indexed, inclusive

	block := strings.Join(lines[startLine:endLine], "\n")
	assert.Contains(t, block, "# helper function")
	assert.Contains(t, block, "def greet(name)")
	assert.Contains(t, block, "# say hello")
	assert.Contains(t, block, "end")
}

func TestStructInfo(t *testing.T) {
	c := &Compiler{}
	src := "struct Dog\n  name\n  breed\nend\n\ndef Dog.bark()\n  return \"woof\"\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Structs, 1)
	assert.Equal(t, "Dog", prog.Structs[0].Name)
	assert.Equal(t, []string{"name", "breed"}, prog.Structs[0].Fields)
	assert.Equal(t, 1, prog.Structs[0].Line)
}

func TestStructInfoMultiple(t *testing.T) {
	c := &Compiler{}
	src := "struct Point\n  x\n  y\nend\n\nstruct Color\n  r\n  g\n  b\nend\n"
	prog, err := c.ParseSource(src, "test.rugo")
	require.NoError(t, err)
	require.Len(t, prog.Structs, 2)
	assert.Equal(t, "Point", prog.Structs[0].Name)
	assert.Equal(t, "Color", prog.Structs[1].Name)
	assert.Equal(t, []string{"r", "g", "b"}, prog.Structs[1].Fields)
}
