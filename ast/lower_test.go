package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLowerSpawnExpr_ExtractsLastExpr(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &SpawnExpr{
				Body: []Statement{
					&ExprStmt{Expression: &CallExpr{Func: &IdentExpr{Name: "puts"}, Args: []Expr{&StringLiteral{Value: "hi"}}}},
					&ExprStmt{Expression: &IntLiteral{Value: "42"}},
				},
			}},
		},
	}

	lowered := Lower(prog)
	require.NotEqual(t, prog, lowered, "should return new program")

	es := lowered.Statements[0].(*ExprStmt)
	ls := es.Expression.(*LoweredSpawnExpr)
	assert.Equal(t, 1, len(ls.Body), "body should have 1 statement (puts call)")
	assert.Equal(t, "42", ls.ResultExpr.(*IntLiteral).Value)
}

func TestLowerSpawnExpr_NoResultExpr(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &SpawnExpr{
				Body: []Statement{
					&AssignStmt{Target: "x", Value: &IntLiteral{Value: "1"}},
				},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	ls := es.Expression.(*LoweredSpawnExpr)
	assert.Equal(t, 1, len(ls.Body))
	assert.Nil(t, ls.ResultExpr)
}

func TestLowerSpawnExpr_EmptyBody(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &SpawnExpr{Body: []Statement{}}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	ls := es.Expression.(*LoweredSpawnExpr)
	assert.Equal(t, 0, len(ls.Body))
	assert.Nil(t, ls.ResultExpr)
}

func TestLowerParallelExpr_CategorizesBranches(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &ParallelExpr{
				Body: []Statement{
					&ExprStmt{Expression: &IntLiteral{Value: "1"}},
					&AssignStmt{Target: "x", Value: &IntLiteral{Value: "2"}},
					&ExprStmt{Expression: &IntLiteral{Value: "3"}},
				},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	lp := es.Expression.(*LoweredParallelExpr)
	require.Equal(t, 3, len(lp.Branches))

	// Branch 0: expression
	assert.NotNil(t, lp.Branches[0].Expr)
	assert.Nil(t, lp.Branches[0].Stmts)
	assert.Equal(t, 0, lp.Branches[0].Index)

	// Branch 1: statement
	assert.Nil(t, lp.Branches[1].Expr)
	assert.Equal(t, 1, len(lp.Branches[1].Stmts))
	assert.Equal(t, 1, lp.Branches[1].Index)

	// Branch 2: expression
	assert.NotNil(t, lp.Branches[2].Expr)
	assert.Equal(t, 2, lp.Branches[2].Index)
}

func TestLowerParallelExpr_Empty(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &ParallelExpr{Body: []Statement{}}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	lp := es.Expression.(*LoweredParallelExpr)
	assert.Equal(t, 0, len(lp.Branches))
}

func TestLowerTryExpr_ExtractsSimpleResult(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &TryExpr{
				Expr:   &CallExpr{Func: &IdentExpr{Name: "risky"}, Args: nil},
				ErrVar: "e",
				Handler: []Statement{
					&ExprStmt{Expression: &CallExpr{Func: &IdentExpr{Name: "puts"}, Args: []Expr{&IdentExpr{Name: "e"}}}},
					&ExprStmt{Expression: &StringLiteral{Value: "fallback"}},
				},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	lt := es.Expression.(*LoweredTryExpr)
	assert.Equal(t, "e", lt.ErrVar)
	assert.Equal(t, 1, len(lt.Handler), "handler should have 1 statement (puts)")
	assert.Equal(t, "fallback", lt.ResultExpr.(*StringLiteral).Value)
}

func TestLowerTryExpr_ComplexResult(t *testing.T) {
	// When last handler statement is IfStmt, ResultExpr should be nil
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &TryExpr{
				Expr:   &IdentExpr{Name: "x"},
				ErrVar: "e",
				Handler: []Statement{
					&IfStmt{
						Condition: &IdentExpr{Name: "e"},
						Body:      []Statement{&ExprStmt{Expression: &IntLiteral{Value: "0"}}},
						ElseBody:  []Statement{&ExprStmt{Expression: &IntLiteral{Value: "1"}}},
					},
				},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	lt := es.Expression.(*LoweredTryExpr)
	assert.Nil(t, lt.ResultExpr, "IfStmt last stmt should not be extracted")
	assert.Equal(t, 1, len(lt.Handler))
}

func TestLowerTryExpr_EmptyHandler(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &TryExpr{
				Expr:    &IdentExpr{Name: "x"},
				ErrVar:  "e",
				Handler: []Statement{},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	lt := es.Expression.(*LoweredTryExpr)
	assert.Nil(t, lt.ResultExpr)
	assert.Equal(t, 0, len(lt.Handler))
}

func TestLowerNested_SpawnInAssign(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&AssignStmt{
				Target: "task",
				Value: &SpawnExpr{
					Body: []Statement{
						&ExprStmt{Expression: &IntLiteral{Value: "99"}},
					},
				},
			},
		},
	}

	lowered := Lower(prog)
	as := lowered.Statements[0].(*AssignStmt)
	ls := as.Value.(*LoweredSpawnExpr)
	assert.Equal(t, "99", ls.ResultExpr.(*IntLiteral).Value)
}

func TestLowerNested_SpawnInFuncDef(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "work",
				Body: []Statement{
					&ReturnStmt{Value: &SpawnExpr{
						Body: []Statement{
							&ExprStmt{Expression: &IntLiteral{Value: "7"}},
						},
					}},
				},
			},
		},
	}

	lowered := Lower(prog)
	fd := lowered.Statements[0].(*FuncDef)
	rs := fd.Body[0].(*ReturnStmt)
	ls := rs.Value.(*LoweredSpawnExpr)
	assert.Equal(t, "7", ls.ResultExpr.(*IntLiteral).Value)
}

func TestLowerNoChange_ReturnsOriginal(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &IntLiteral{Value: "42"}},
		},
	}

	lowered := Lower(prog)
	assert.Same(t, prog, lowered, "should return same pointer when no changes")
}

func TestLowerNested_SpawnInLambda(t *testing.T) {
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{Expression: &FnExpr{
				Body: []Statement{
					&ExprStmt{Expression: &SpawnExpr{
						Body: []Statement{
							&ExprStmt{Expression: &IntLiteral{Value: "5"}},
						},
					}},
				},
			}},
		},
	}

	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	fn := es.Expression.(*FnExpr)
	inner := fn.Body[0].(*ExprStmt)
	ls := inner.Expression.(*LoweredSpawnExpr)
	assert.Equal(t, "5", ls.ResultExpr.(*IntLiteral).Value)
}

func TestLowerReturnInSpawn(t *testing.T) {
	// return inside spawn body → SpawnReturnStmt
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &SpawnExpr{
					Body: []Statement{
						&ReturnStmt{BaseStmt: BaseStmt{SourceLine: 2}, Value: &IntLiteral{Value: "42"}},
					},
				},
			},
		},
	}
	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	spawn := es.Expression.(*LoweredSpawnExpr)
	// The return was the only statement and it was also the last → extracted as ResultExpr?
	// No — ReturnStmt is not an ExprStmt, so it stays in Body as SpawnReturnStmt
	require.Len(t, spawn.Body, 1)
	sr, ok := spawn.Body[0].(*SpawnReturnStmt)
	require.True(t, ok, "should be SpawnReturnStmt, got %T", spawn.Body[0])
	assert.Equal(t, "42", sr.Value.(*IntLiteral).Value)
	assert.Equal(t, 2, sr.StmtLine())
}

func TestLowerReturnInTryHandler(t *testing.T) {
	// return inside try handler → TryHandlerReturnStmt
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &TryExpr{
					Expr:   &IntLiteral{Value: "1"},
					ErrVar: "err",
					Handler: []Statement{
						&ReturnStmt{BaseStmt: BaseStmt{SourceLine: 3}, Value: &IntLiteral{Value: "fallback"}},
					},
				},
			},
		},
	}
	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	tryExpr := es.Expression.(*LoweredTryExpr)
	require.Len(t, tryExpr.Handler, 1)
	tr, ok := tryExpr.Handler[0].(*TryHandlerReturnStmt)
	require.True(t, ok, "should be TryHandlerReturnStmt, got %T", tryExpr.Handler[0])
	assert.Equal(t, "fallback", tr.Value.(*IntLiteral).Value)
	assert.Equal(t, 3, tr.StmtLine())
}

func TestLowerReturnInNormalFunc(t *testing.T) {
	// return in normal function → unchanged ReturnStmt
	prog := &Program{
		Statements: []Statement{
			&FuncDef{
				Name: "foo",
				Body: []Statement{
					&ReturnStmt{Value: &IntLiteral{Value: "1"}},
				},
			},
		},
	}
	lowered := Lower(prog)
	fd := lowered.Statements[0].(*FuncDef)
	_, ok := fd.Body[0].(*ReturnStmt)
	assert.True(t, ok, "normal function return stays as ReturnStmt")
}

func TestLowerBareReturnInSpawn(t *testing.T) {
	// bare return (no value) inside spawn → SpawnReturnStmt with nil Value
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &SpawnExpr{
					Body: []Statement{
						&ExprStmt{Expression: &IntLiteral{Value: "1"}},
						&ReturnStmt{BaseStmt: BaseStmt{SourceLine: 3}},
					},
				},
			},
		},
	}
	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	spawn := es.Expression.(*LoweredSpawnExpr)
	// Last stmt is SpawnReturnStmt (not ExprStmt), so no ResultExpr extraction
	assert.Nil(t, spawn.ResultExpr, "no ResultExpr because last stmt is SpawnReturnStmt")
	require.Len(t, spawn.Body, 2)
	_, ok := spawn.Body[0].(*ExprStmt)
	assert.True(t, ok, "first stmt unchanged")
	sr, ok := spawn.Body[1].(*SpawnReturnStmt)
	require.True(t, ok, "should be SpawnReturnStmt")
	assert.Nil(t, sr.Value, "bare return has nil value")
}

func TestLowerReturnInsideIfInsideSpawn(t *testing.T) {
	// return inside if inside spawn → SpawnReturnStmt (recursive descent)
	prog := &Program{
		Statements: []Statement{
			&ExprStmt{
				Expression: &SpawnExpr{
					Body: []Statement{
						&IfStmt{
							Condition: &BoolLiteral{Value: true},
							Body: []Statement{
								&ReturnStmt{Value: &IntLiteral{Value: "99"}},
							},
						},
					},
				},
			},
		},
	}
	lowered := Lower(prog)
	es := lowered.Statements[0].(*ExprStmt)
	spawn := es.Expression.(*LoweredSpawnExpr)
	ifStmt := spawn.Body[0].(*IfStmt)
	sr, ok := ifStmt.Body[0].(*SpawnReturnStmt)
	require.True(t, ok, "return inside if inside spawn should be SpawnReturnStmt")
	assert.Equal(t, "99", sr.Value.(*IntLiteral).Value)
}
