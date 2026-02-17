package ast

import (
	"github.com/rubiojr/rugo/preprocess"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactoryLoweredSpawn(t *testing.T) {
	f := NewFactory()
	body := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}}
	result := &IntLiteral{Value: "42"}

	node := f.LoweredSpawn(body, result)
	require.NotNil(t, node)
	assert.Equal(t, body, node.Body)
	assert.Equal(t, result, node.ResultExpr)
}

func TestFactoryLoweredSpawnNilResult(t *testing.T) {
	f := NewFactory()
	body := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}}

	node := f.LoweredSpawn(body, nil)
	assert.Nil(t, node.ResultExpr)
	assert.Equal(t, body, node.Body)
}

func TestFactoryLoweredParallel(t *testing.T) {
	f := NewFactory()
	branches := []ParallelBranch{
		f.ParallelBranchExpr(&IntLiteral{Value: "1"}, 0),
		f.ParallelBranchStmts([]Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}, 1),
	}

	node := f.LoweredParallel(branches)
	require.Len(t, node.Branches, 2)
	assert.NotNil(t, node.Branches[0].Expr)
	assert.Equal(t, 0, node.Branches[0].Index)
	assert.NotNil(t, node.Branches[1].Stmts)
	assert.Equal(t, 1, node.Branches[1].Index)
}

func TestFactoryLoweredTry(t *testing.T) {
	f := NewFactory()
	expr := &IntLiteral{Value: "1"}
	handler := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}
	result := &IntLiteral{Value: "3"}

	node := f.LoweredTry(expr, "err", handler, result)
	assert.Equal(t, expr, node.Expr)
	assert.Equal(t, "err", node.ErrVar)
	assert.Equal(t, handler, node.Handler)
	assert.Equal(t, result, node.ResultExpr)
}

func TestFactoryLoweredTryNilResult(t *testing.T) {
	f := NewFactory()
	node := f.LoweredTry(&IntLiteral{Value: "1"}, "e", nil, nil)
	assert.Nil(t, node.Handler)
	assert.Nil(t, node.ResultExpr)
}

func TestFactoryProgramFrom(t *testing.T) {
	f := NewFactory()
	src := &Program{
		SourceFile: "test.rugo",
		RawSource:  "source code",
		Structs:    []preprocess.StructInfo{{Name: "Dog"}},
		Statements: []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}},
	}
	newStmts := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}

	result := f.ProgramFrom(src, newStmts)
	assert.Equal(t, newStmts, result.Statements)
	assert.Equal(t, src.SourceFile, result.SourceFile)
	assert.Equal(t, src.RawSource, result.RawSource)
	assert.Equal(t, src.Structs, result.Structs)
}

func TestFactoryFuncDefWithBody(t *testing.T) {
	f := NewFactory()
	src := &FuncDef{
		Name:   "foo",
		Params: []Param{{Name: "x"}},
		Body:   []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}},
	}
	newBody := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}

	result := f.FuncDefWithBody(src, newBody)
	assert.Equal(t, "foo", result.Name)
	assert.Equal(t, src.Params, result.Params)
	assert.Equal(t, newBody, result.Body)
	assert.NotSame(t, src, result)
}

func TestFactoryIfStmtWithBranches(t *testing.T) {
	f := NewFactory()
	cond := &BoolLiteral{Value: true}
	src := &IfStmt{
		Condition: cond,
		Body:      []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}},
	}
	newBody := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}
	newElse := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "3"}}}
	newElsifs := []ElsifClause{{Condition: cond, Body: newBody}}

	result := f.IfStmtWithBranches(src, newBody, newElse, newElsifs)
	assert.Equal(t, cond, result.Condition)
	assert.Equal(t, newBody, result.Body)
	assert.Equal(t, newElse, result.ElseBody)
	assert.Equal(t, newElsifs, result.ElsifClauses)
	assert.NotSame(t, src, result)
}

func TestFactoryFnExprWithBody(t *testing.T) {
	f := NewFactory()
	src := &FnExpr{
		Params: []Param{{Name: "x"}},
		Body:   []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}},
	}
	newBody := []Statement{&ExprStmt{Expression: &IntLiteral{Value: "2"}}}

	result := f.FnExprWithBody(src, newBody)
	assert.Equal(t, src.Params, result.Params)
	assert.Equal(t, newBody, result.Body)
}
