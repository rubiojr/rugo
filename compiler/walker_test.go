package compiler

import (
	"github.com/rubiojr/rugo/ast"
	"strings"
	"testing"

	"github.com/rubiojr/rugo/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper to parse and walk a rugo source string into a typed AST.
func parseAndWalk(t *testing.T, src string) *ast.Program {
	t.Helper()
	cleaned, err := ast.StripComments(src)
	if err != nil {
		t.Fatalf("stripComments error: %v", err)
	}
	if !strings.HasSuffix(cleaned, "\n") {
		cleaned += "\n"
	}
	p := &parser.Parser{}
	flatAST, err := p.Parse("test.rugo", []byte(cleaned))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	prog, err := ast.Walk(p, flatAST)
	if err != nil {
		t.Fatalf("Walk error: %v", err)
	}
	return prog
}

func TestWalkAssignment(t *testing.T) {
	prog := parseAndWalk(t, `x = 42`)
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	assign, ok := prog.Statements[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("expected ast.AssignStmt, got %T", prog.Statements[0])
	}
	if assign.Target != "x" {
		t.Errorf("target = %q, want %q", assign.Target, "x")
	}
	lit, ok := assign.Value.(*ast.IntLiteral)
	if !ok {
		t.Fatalf("expected ast.IntLiteral, got %T", assign.Value)
	}
	if lit.Value != "42" {
		t.Errorf("value = %q, want %q", lit.Value, "42")
	}
}

func TestWalkStringAssignment(t *testing.T) {
	prog := parseAndWalk(t, `x = "hello"`)
	assign, ok := prog.Statements[0].(*ast.AssignStmt)
	if !ok {
		t.Fatalf("expected ast.AssignStmt, got %T", prog.Statements[0])
	}
	lit, ok := assign.Value.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected ast.StringLiteral, got %T", assign.Value)
	}
	if lit.Value != "hello" {
		t.Errorf("value = %q, want %q", lit.Value, "hello")
	}
}

func TestWalkFloat(t *testing.T) {
	prog := parseAndWalk(t, `x = 3.14`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	lit, ok := assign.Value.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("expected ast.FloatLiteral, got %T", assign.Value)
	}
	if lit.Value != "3.14" {
		t.Errorf("value = %q, want %q", lit.Value, "3.14")
	}
}

func TestWalkBooleans(t *testing.T) {
	prog := parseAndWalk(t, "x = true\ny = false")
	assign1 := prog.Statements[0].(*ast.AssignStmt)
	assign2 := prog.Statements[1].(*ast.AssignStmt)
	b1 := assign1.Value.(*ast.BoolLiteral)
	b2 := assign2.Value.(*ast.BoolLiteral)
	if !b1.Value {
		t.Error("expected true")
	}
	if b2.Value {
		t.Error("expected false")
	}
}

func TestWalkNil(t *testing.T) {
	prog := parseAndWalk(t, `x = nil`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	_, ok := assign.Value.(*ast.NilLiteral)
	if !ok {
		t.Fatalf("expected ast.NilLiteral, got %T", assign.Value)
	}
}

func TestWalkBinaryExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = 1 + 2`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	bin, ok := assign.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected ast.BinaryExpr, got %T", assign.Value)
	}
	if bin.Op != "+" {
		t.Errorf("op = %q, want %q", bin.Op, "+")
	}
}

func TestWalkUnaryExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = -1`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	unary, ok := assign.Value.(*ast.UnaryExpr)
	if !ok {
		t.Fatalf("expected ast.UnaryExpr, got %T", assign.Value)
	}
	if unary.Op != "-" {
		t.Errorf("op = %q, want %q", unary.Op, "-")
	}
}

func TestWalkNotExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = !true`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	unary, ok := assign.Value.(*ast.UnaryExpr)
	if !ok {
		t.Fatalf("expected ast.UnaryExpr, got %T", assign.Value)
	}
	if unary.Op != "!" {
		t.Errorf("op = %q, want %q", unary.Op, "!")
	}
}

func TestWalkComparison(t *testing.T) {
	ops := []string{"==", "!=", "<", ">", "<=", ">="}
	for _, op := range ops {
		prog := parseAndWalk(t, "x = 1 "+op+" 2")
		assign := prog.Statements[0].(*ast.AssignStmt)
		bin, ok := assign.Value.(*ast.BinaryExpr)
		if !ok {
			t.Fatalf("[%s] expected ast.BinaryExpr, got %T", op, assign.Value)
		}
		if bin.Op != op {
			t.Errorf("op = %q, want %q", bin.Op, op)
		}
	}
}

func TestWalkFuncCall(t *testing.T) {
	prog := parseAndWalk(t, `puts("hello", "world")`)
	exprStmt, ok := prog.Statements[0].(*ast.ExprStmt)
	if !ok {
		t.Fatalf("expected ast.ExprStmt, got %T", prog.Statements[0])
	}
	call, ok := exprStmt.Expression.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected ast.CallExpr, got %T", exprStmt.Expression)
	}
	ident, ok := call.Func.(*ast.IdentExpr)
	if !ok {
		t.Fatalf("expected ast.IdentExpr, got %T", call.Func)
	}
	if ident.Name != "puts" {
		t.Errorf("func name = %q, want %q", ident.Name, "puts")
	}
	if len(call.Args) != 2 {
		t.Errorf("args count = %d, want 2", len(call.Args))
	}
}

func TestWalkFuncDef(t *testing.T) {
	prog := parseAndWalk(t, "def greet(name)\nputs(name)\nend")
	funcDef, ok := prog.Statements[0].(*ast.FuncDef)
	if !ok {
		t.Fatalf("expected ast.FuncDef, got %T", prog.Statements[0])
	}
	if funcDef.Name != "greet" {
		t.Errorf("name = %q, want %q", funcDef.Name, "greet")
	}
	if len(funcDef.Params) != 1 || funcDef.Params[0].Name != "name" {
		t.Errorf("params = %v, want [name]", funcDef.Params)
	}
	if len(funcDef.Body) != 1 {
		t.Errorf("body length = %d, want 1", len(funcDef.Body))
	}
}

func TestWalkFuncDefNoParams(t *testing.T) {
	prog := parseAndWalk(t, "def hello()\nputs(\"hi\")\nend")
	funcDef := prog.Statements[0].(*ast.FuncDef)
	if len(funcDef.Params) != 0 {
		t.Errorf("params = %v, want []", funcDef.Params)
	}
}

func TestWalkIfStmt(t *testing.T) {
	prog := parseAndWalk(t, "if x == 1\nputs(\"one\")\nelsif x == 2\nputs(\"two\")\nelse\nputs(\"other\")\nend")
	ifStmt, ok := prog.Statements[0].(*ast.IfStmt)
	if !ok {
		t.Fatalf("expected ast.IfStmt, got %T", prog.Statements[0])
	}
	if len(ifStmt.ElsifClauses) != 1 {
		t.Errorf("elsif clauses = %d, want 1", len(ifStmt.ElsifClauses))
	}
	if len(ifStmt.ElseBody) != 1 {
		t.Errorf("else body length = %d, want 1", len(ifStmt.ElseBody))
	}
}

func TestWalkWhileStmt(t *testing.T) {
	prog := parseAndWalk(t, "while x > 0\nx = x - 1\nend")
	whileStmt, ok := prog.Statements[0].(*ast.WhileStmt)
	if !ok {
		t.Fatalf("expected ast.WhileStmt, got %T", prog.Statements[0])
	}
	if len(whileStmt.Body) != 1 {
		t.Errorf("body length = %d, want 1", len(whileStmt.Body))
	}
}

func TestWalkReturnStmt(t *testing.T) {
	prog := parseAndWalk(t, "def foo()\nreturn 42\nend")
	funcDef := prog.Statements[0].(*ast.FuncDef)
	retStmt, ok := funcDef.Body[0].(*ast.ReturnStmt)
	if !ok {
		t.Fatalf("expected ast.ReturnStmt, got %T", funcDef.Body[0])
	}
	if retStmt.Value == nil {
		t.Fatal("expected return value")
	}
}

func TestWalkBareReturn(t *testing.T) {
	prog := parseAndWalk(t, "def foo()\nreturn\nend")
	funcDef := prog.Statements[0].(*ast.FuncDef)
	retStmt := funcDef.Body[0].(*ast.ReturnStmt)
	if retStmt.Value != nil {
		t.Fatal("expected bare return (nil value)")
	}
}

func TestWalkRequireStmt(t *testing.T) {
	prog := parseAndWalk(t, `require "helpers"`)
	req, ok := prog.Statements[0].(*ast.RequireStmt)
	if !ok {
		t.Fatalf("expected ast.RequireStmt, got %T", prog.Statements[0])
	}
	if req.Path != "helpers" {
		t.Errorf("path = %q, want %q", req.Path, "helpers")
	}
}

func TestWalkArrayLiteral(t *testing.T) {
	prog := parseAndWalk(t, `x = [1, 2, 3]`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	arr, ok := assign.Value.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ast.ArrayLiteral, got %T", assign.Value)
	}
	if len(arr.Elements) != 3 {
		t.Errorf("elements = %d, want 3", len(arr.Elements))
	}
}

func TestWalkEmptyArray(t *testing.T) {
	prog := parseAndWalk(t, `x = []`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	arr := assign.Value.(*ast.ArrayLiteral)
	if len(arr.Elements) != 0 {
		t.Errorf("elements = %d, want 0", len(arr.Elements))
	}
}

func TestWalkHashLiteral(t *testing.T) {
	prog := parseAndWalk(t, `x = {"a" => 1, "b" => 2}`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	hash, ok := assign.Value.(*ast.HashLiteral)
	if !ok {
		t.Fatalf("expected ast.HashLiteral, got %T", assign.Value)
	}
	if len(hash.Pairs) != 2 {
		t.Errorf("pairs = %d, want 2", len(hash.Pairs))
	}
}

func TestWalkIndexExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = arr[0]`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	idx, ok := assign.Value.(*ast.IndexExpr)
	if !ok {
		t.Fatalf("expected ast.IndexExpr, got %T", assign.Value)
	}
	ident := idx.Object.(*ast.IdentExpr)
	if ident.Name != "arr" {
		t.Errorf("object = %q, want %q", ident.Name, "arr")
	}
}

func TestWalkNestedExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = (1 + 2) * 3`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	mul, ok := assign.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected ast.BinaryExpr, got %T", assign.Value)
	}
	if mul.Op != "*" {
		t.Errorf("outer op = %q, want %q", mul.Op, "*")
	}
	add, ok := mul.Left.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected inner ast.BinaryExpr, got %T", mul.Left)
	}
	if add.Op != "+" {
		t.Errorf("inner op = %q, want %q", add.Op, "+")
	}
}

func TestWalkMultipleStatements(t *testing.T) {
	prog := parseAndWalk(t, "x = 1\ny = 2\nputs(x + y)")
	if len(prog.Statements) != 3 {
		t.Fatalf("expected 3 statements, got %d", len(prog.Statements))
	}
}

func TestWalkBooleanOps(t *testing.T) {
	prog := parseAndWalk(t, `x = a && b || c`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	or, ok := assign.Value.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected ast.BinaryExpr (||), got %T", assign.Value)
	}
	if or.Op != "||" {
		t.Errorf("outer op = %q, want ||", or.Op)
	}
	and, ok := or.Left.(*ast.BinaryExpr)
	if !ok {
		t.Fatalf("expected ast.BinaryExpr (&&), got %T", or.Left)
	}
	if and.Op != "&&" {
		t.Errorf("inner op = %q, want &&", and.Op)
	}
}

func TestWalkRequireWithAlias(t *testing.T) {
	prog := parseAndWalk(t, `require "helpers" as "h"`)
	req, ok := prog.Statements[0].(*ast.RequireStmt)
	if !ok {
		t.Fatalf("expected ast.RequireStmt, got %T", prog.Statements[0])
	}
	if req.Path != "helpers" {
		t.Errorf("path = %q, want %q", req.Path, "helpers")
	}
	if req.Alias != "h" {
		t.Errorf("alias = %q, want %q", req.Alias, "h")
	}
}

func TestWalkRequireWithSingle(t *testing.T) {
	prog := parseAndWalk(t, `require "github.com/user/repo" with client`)
	req, ok := prog.Statements[0].(*ast.RequireStmt)
	if !ok {
		t.Fatalf("expected ast.RequireStmt, got %T", prog.Statements[0])
	}
	if req.Path != "github.com/user/repo" {
		t.Errorf("path = %q, want %q", req.Path, "github.com/user/repo")
	}
	if req.Alias != "" {
		t.Errorf("alias = %q, want empty", req.Alias)
	}
	if len(req.With) != 1 || req.With[0] != "client" {
		t.Errorf("with = %v, want [client]", req.With)
	}
}

func TestWalkRequireWithMultiple(t *testing.T) {
	prog := parseAndWalk(t, `require "github.com/user/repo" with client, issue, repo`)
	req, ok := prog.Statements[0].(*ast.RequireStmt)
	if !ok {
		t.Fatalf("expected ast.RequireStmt, got %T", prog.Statements[0])
	}
	if len(req.With) != 3 {
		t.Fatalf("with = %v, want 3 items", req.With)
	}
	if req.With[0] != "client" || req.With[1] != "issue" || req.With[2] != "repo" {
		t.Errorf("with = %v, want [client issue repo]", req.With)
	}
}

func TestWalkDotExpr(t *testing.T) {
	prog := parseAndWalk(t, `x = ns.value`)
	assign := prog.Statements[0].(*ast.AssignStmt)
	dot, ok := assign.Value.(*ast.DotExpr)
	if !ok {
		t.Fatalf("expected ast.DotExpr, got %T", assign.Value)
	}
	if dot.Field != "value" {
		t.Errorf("field = %q, want %q", dot.Field, "value")
	}
	ident := dot.Object.(*ast.IdentExpr)
	if ident.Name != "ns" {
		t.Errorf("object = %q, want %q", ident.Name, "ns")
	}
}

func TestWalkDotCall(t *testing.T) {
	prog := parseAndWalk(t, `ns.func(1, 2)`)
	exprStmt := prog.Statements[0].(*ast.ExprStmt)
	call, ok := exprStmt.Expression.(*ast.CallExpr)
	if !ok {
		t.Fatalf("expected ast.CallExpr, got %T", exprStmt.Expression)
	}
	dot, ok := call.Func.(*ast.DotExpr)
	if !ok {
		t.Fatalf("expected ast.DotExpr as call func, got %T", call.Func)
	}
	if dot.Field != "func" {
		t.Errorf("field = %q, want %q", dot.Field, "func")
	}
	if len(call.Args) != 2 {
		t.Errorf("args = %d, want 2", len(call.Args))
	}
}

func TestWalkUseStmt(t *testing.T) {
	prog := parseAndWalk(t, `use "http"`)
	use, ok := prog.Statements[0].(*ast.UseStmt)
	if !ok {
		t.Fatalf("expected ast.UseStmt, got %T", prog.Statements[0])
	}
	if use.Module != "http" {
		t.Errorf("module = %q, want %q", use.Module, "http")
	}
}

func TestWalkTryExpr(t *testing.T) {
	prog := parseAndWalk(t, `use "os"`+"\n"+`x = try os.exec("ls") or err`+"\n"+`"fallback"`+"\n"+`end`)
	found := false
	for _, s := range prog.Statements {
		if assign, ok := s.(*ast.AssignStmt); ok {
			if _, ok := assign.Value.(*ast.TryExpr); ok {
				found = true
			}
		}
	}
	if !found {
		t.Error("expected to find ast.TryExpr in AST")
	}
}

func TestWalkForIn(t *testing.T) {
	prog := parseAndWalk(t, "for x in arr\nputs(x)\nend\n")
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	forStmt, ok := prog.Statements[0].(*ast.ForStmt)
	if !ok {
		t.Fatalf("expected ast.ForStmt, got %T", prog.Statements[0])
	}
	if forStmt.Var != "x" {
		t.Errorf("expected Var='x', got %q", forStmt.Var)
	}
	if forStmt.IndexVar != "" {
		t.Errorf("expected IndexVar='', got %q", forStmt.IndexVar)
	}
}

func TestWalkForInWithIndex(t *testing.T) {
	prog := parseAndWalk(t, "for i, x in arr\nputs(x)\nend\n")
	forStmt := prog.Statements[0].(*ast.ForStmt)
	if forStmt.Var != "i" || forStmt.IndexVar != "x" {
		t.Errorf("expected Var='i', IndexVar='x', got %q, %q", forStmt.Var, forStmt.IndexVar)
	}
}

func TestWalkBreak(t *testing.T) {
	prog := parseAndWalk(t, "while true\nbreak\nend\n")
	whileStmt := prog.Statements[0].(*ast.WhileStmt)
	_, ok := whileStmt.Body[0].(*ast.BreakStmt)
	if !ok {
		t.Fatalf("expected ast.BreakStmt, got %T", whileStmt.Body[0])
	}
}

func TestWalkNext(t *testing.T) {
	prog := parseAndWalk(t, "while true\nnext\nend\n")
	whileStmt := prog.Statements[0].(*ast.WhileStmt)
	_, ok := whileStmt.Body[0].(*ast.NextStmt)
	if !ok {
		t.Fatalf("expected ast.NextStmt, got %T", whileStmt.Body[0])
	}
}

func TestWalkIndexAssign(t *testing.T) {
	prog := parseAndWalk(t, `arr[0] = 42`+"\n")
	if len(prog.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(prog.Statements))
	}
	ia, ok := prog.Statements[0].(*ast.IndexAssignStmt)
	if !ok {
		t.Fatalf("expected ast.IndexAssignStmt, got %T", prog.Statements[0])
	}
	obj, ok := ia.Object.(*ast.IdentExpr)
	if !ok || obj.Name != "arr" {
		t.Errorf("expected object 'arr', got %v", ia.Object)
	}
}

func TestWalkHashAssign(t *testing.T) {
	prog := parseAndWalk(t, `h["key"] = "val"`+"\n")
	_, ok := prog.Statements[0].(*ast.IndexAssignStmt)
	if !ok {
		t.Fatalf("expected ast.IndexAssignStmt, got %T", prog.Statements[0])
	}
}

// --- New walker tests for uncovered node types ---

func TestWalkTestDef(t *testing.T) {
	prog := parseAndWalk(t, "rats \"my test\"\n  puts(\"hello\")\nend\n")
	require.Len(t, prog.Statements, 1)
	td, ok := prog.Statements[0].(*ast.TestDef)
	require.True(t, ok, "expected *ast.TestDef, got %T", prog.Statements[0])
	assert.Equal(t, "my test", td.Name)
	assert.NotEmpty(t, td.Body, "expected at least one body statement")
}

func TestWalkSpawnExpr(t *testing.T) {
	prog := parseAndWalk(t, "x = spawn\n  1 + 2\nend\n")
	require.Len(t, prog.Statements, 1)
	assign, ok := prog.Statements[0].(*ast.AssignStmt)
	require.True(t, ok, "expected *ast.AssignStmt, got %T", prog.Statements[0])
	spawn, ok := assign.Value.(*ast.SpawnExpr)
	require.True(t, ok, "expected *ast.SpawnExpr, got %T", assign.Value)
	assert.NotEmpty(t, spawn.Body, "expected at least one body statement")
}

func TestWalkParallelExpr(t *testing.T) {
	prog := parseAndWalk(t, "x = parallel\n  1\n  2\nend\n")
	require.Len(t, prog.Statements, 1)
	assign, ok := prog.Statements[0].(*ast.AssignStmt)
	require.True(t, ok, "expected *ast.AssignStmt, got %T", prog.Statements[0])
	par, ok := assign.Value.(*ast.ParallelExpr)
	require.True(t, ok, "expected *ast.ParallelExpr, got %T", assign.Value)
	assert.NotEmpty(t, par.Body, "expected at least one body statement")
}
