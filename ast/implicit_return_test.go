package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImplicitReturnFuncSimple(t *testing.T) {
	// def foo() 42 end → last ExprStmt becomes ImplicitReturnStmt
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&ExprStmt{BaseStmt: BaseStmt{SourceLine: 1}, Expression: &IntLiteral{Value: "42"}},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	fd := result.Statements[0].(*FuncDef)
	require.Len(t, fd.Body, 1)
	ret, ok := fd.Body[0].(*ImplicitReturnStmt)
	require.True(t, ok, "last stmt should be ImplicitReturnStmt")
	assert.Equal(t, "42", ret.Value.(*IntLiteral).Value)
	assert.Equal(t, 1, ret.StmtLine())
}

func TestImplicitReturnFuncMultiStmt(t *testing.T) {
	// def foo() x = 1; x + 2 end → only last becomes ImplicitReturnStmt
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&AssignStmt{Target: "x", Value: &IntLiteral{Value: "1"}},
					&ExprStmt{BaseStmt: BaseStmt{SourceLine: 2}, Expression: &IntLiteral{Value: "42"}},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	fd := result.Statements[0].(*FuncDef)
	require.Len(t, fd.Body, 2)
	_, ok := fd.Body[0].(*AssignStmt)
	assert.True(t, ok, "first stmt unchanged")
	ret, ok := fd.Body[1].(*ImplicitReturnStmt)
	require.True(t, ok, "last stmt should be ImplicitReturnStmt")
	assert.Equal(t, "42", ret.Value.(*IntLiteral).Value)
}

func TestImplicitReturnFuncIfStmt(t *testing.T) {
	// def foo() if cond; 1; else; 2; end; end
	// → ImplicitReturnStmt in each branch
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&IfStmt{
						Condition: &BoolLiteral{Value: true},
						Body: []Statement{
							&ExprStmt{BaseStmt: BaseStmt{SourceLine: 2}, Expression: &IntLiteral{Value: "1"}},
						},
						ElseBody: []Statement{
							&ExprStmt{BaseStmt: BaseStmt{SourceLine: 4}, Expression: &IntLiteral{Value: "2"}},
						},
					},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	fd := result.Statements[0].(*FuncDef)
	require.Len(t, fd.Body, 1)
	ifStmt := fd.Body[0].(*IfStmt)

	ret1, ok := ifStmt.Body[0].(*ImplicitReturnStmt)
	require.True(t, ok, "if body last should be ImplicitReturnStmt")
	assert.Equal(t, "1", ret1.Value.(*IntLiteral).Value)

	ret2, ok := ifStmt.ElseBody[0].(*ImplicitReturnStmt)
	require.True(t, ok, "else body last should be ImplicitReturnStmt")
	assert.Equal(t, "2", ret2.Value.(*IntLiteral).Value)
}

func TestImplicitReturnFuncIfElsifStmt(t *testing.T) {
	// if cond1; 1; elsif cond2; 2; else; 3; end
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&IfStmt{
						Condition: &BoolLiteral{Value: true},
						Body: []Statement{
							&ExprStmt{Expression: &IntLiteral{Value: "1"}},
						},
						ElsifClauses: []ElsifClause{
							{
								Condition: &BoolLiteral{Value: false},
								Body: []Statement{
									&ExprStmt{Expression: &IntLiteral{Value: "2"}},
								},
							},
						},
						ElseBody: []Statement{
							&ExprStmt{Expression: &IntLiteral{Value: "3"}},
						},
					},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	fd := result.Statements[0].(*FuncDef)
	ifStmt := fd.Body[0].(*IfStmt)

	_, ok := ifStmt.Body[0].(*ImplicitReturnStmt)
	assert.True(t, ok, "if body")
	_, ok = ifStmt.ElsifClauses[0].Body[0].(*ImplicitReturnStmt)
	assert.True(t, ok, "elsif body")
	_, ok = ifStmt.ElseBody[0].(*ImplicitReturnStmt)
	assert.True(t, ok, "else body")
}

func TestImplicitReturnLambda(t *testing.T) {
	// fn(x) x + 1 end → ImplicitReturnStmt
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &FnExpr{
					Params: []Param{{Name: "x"}},
					Body: []Statement{
						&ExprStmt{BaseStmt: BaseStmt{SourceLine: 1}, Expression: &IntLiteral{Value: "42"}},
					},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	es := result.Statements[0].(*ExprStmt)
	fn := es.Expression.(*FnExpr)
	require.Len(t, fn.Body, 1)
	_, ok := fn.Body[0].(*ImplicitReturnStmt)
	assert.True(t, ok, "lambda last should be ImplicitReturnStmt")
}

func TestImplicitReturnTryHandler(t *testing.T) {
	// try handler with IfStmt result (no ResultExpr) → TryResultStmt in branches
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &LoweredTryExpr{
					Expr:   &IntLiteral{Value: "1"},
					ErrVar: "err",
					Handler: []Statement{
						&IfStmt{
							Condition: &BoolLiteral{Value: true},
							Body: []Statement{
								&ExprStmt{Expression: &IntLiteral{Value: "10"}},
							},
							ElseBody: []Statement{
								&ExprStmt{Expression: &IntLiteral{Value: "20"}},
							},
						},
					},
					ResultExpr: nil, // complex case
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	es := result.Statements[0].(*ExprStmt)
	tryExpr := es.Expression.(*LoweredTryExpr)
	require.Nil(t, tryExpr.ResultExpr, "ResultExpr stays nil")

	ifStmt := tryExpr.Handler[0].(*IfStmt)
	ret1, ok := ifStmt.Body[0].(*TryResultStmt)
	require.True(t, ok, "if body should be TryResultStmt")
	assert.Equal(t, "10", ret1.Value.(*IntLiteral).Value)

	ret2, ok := ifStmt.ElseBody[0].(*TryResultStmt)
	require.True(t, ok, "else body should be TryResultStmt")
	assert.Equal(t, "20", ret2.Value.(*IntLiteral).Value)
}

func TestImplicitReturnTryHandlerSimpleExpr(t *testing.T) {
	// try handler with simple ExprStmt result (no IfStmt) → TryResultStmt
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &LoweredTryExpr{
					Expr:   &IntLiteral{Value: "1"},
					ErrVar: "err",
					Handler: []Statement{
						&ExprStmt{Expression: &IntLiteral{Value: "fallback"}},
					},
					ResultExpr: nil,
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	es := result.Statements[0].(*ExprStmt)
	tryExpr := es.Expression.(*LoweredTryExpr)
	ret, ok := tryExpr.Handler[0].(*TryResultStmt)
	require.True(t, ok, "handler last should be TryResultStmt")
	assert.Equal(t, "fallback", ret.Value.(*IntLiteral).Value)
}

func TestImplicitReturnTryWithResultExprUnchanged(t *testing.T) {
	// try handler with ResultExpr already set → no transformation of handler
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &LoweredTryExpr{
					Expr:       &IntLiteral{Value: "1"},
					ErrVar:     "err",
					Handler:    []Statement{},
					ResultExpr: &IntLiteral{Value: "42"},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	// No transformation needed — program unchanged
	assert.Same(t, prog, result)
}

func TestImplicitReturnEmptyBody(t *testing.T) {
	// def foo() end → no transformation
	prog := &Program{
		Statements: []Statement{
			&FuncDef{Name: "foo", Body: []Statement{}},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	assert.Same(t, prog, result, "empty body unchanged")
}

func TestImplicitReturnNonExprLast(t *testing.T) {
	// def foo() while true; end; end → no implicit return (last is WhileStmt)
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&WhileStmt{
						Condition: &BoolLiteral{Value: true},
						Body:      []Statement{},
					},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	assert.Same(t, prog, result, "non-expr last stmt unchanged")
}

func TestImplicitReturnNestedLambdaInFunc(t *testing.T) {
	// def foo() fn(x) x end end → both get implicit return
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&ExprStmt{
						Expression: &FnExpr{
							Params: []Param{{Name: "x"}},
							Body: []Statement{
								&ExprStmt{Expression: &IdentExpr{Name: "x"}},
							},
						},
					},
				},
			},
		},
	}
	result := ImplicitReturnLowering().Transform(prog)
	fd := result.Statements[0].(*FuncDef)
	require.Len(t, fd.Body, 1)

	// Outer function: last stmt is ImplicitReturnStmt wrapping the FnExpr
	outerRet := fd.Body[0].(*ImplicitReturnStmt)
	fn := outerRet.Value.(*FnExpr)

	// Inner lambda: body has ImplicitReturnStmt
	innerRet, ok := fn.Body[0].(*ImplicitReturnStmt)
	require.True(t, ok, "lambda body should have ImplicitReturnStmt")
	assert.Equal(t, "x", innerRet.Value.(*IdentExpr).Name)
}
