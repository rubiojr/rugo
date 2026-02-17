package compiler

import (
	"github.com/rubiojr/rugo/ast"
	"testing"

	"github.com/rubiojr/rugo/modules"
)

func TestWalkExpressionsFindsSpawn(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.AssignStmt{Target: "t", Value: &ast.LoweredSpawnExpr{
				Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
			}},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn")
	}
	if astUsesParallel(prog) {
		t.Error("expected astUsesParallel to be false")
	}
}

func TestWalkExpressionsFindsParallel(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.LoweredParallelExpr{
				Branches: []ast.ParallelBranch{{Expr: &ast.IntLiteral{Value: "1"}, Index: 0}},
			}},
		},
	}
	if !astUsesParallel(prog) {
		t.Error("expected astUsesParallel to find parallel")
	}
	if astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to be false")
	}
}

func TestWalkExpressionsFindsTaskMethods(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.DotExpr{
				Object: &ast.IdentExpr{Name: "task"},
				Field:  "value",
			}},
		},
	}
	if !astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to find .value")
	}
}

func TestWalkExpressionsTaskMethodOnModuleIgnored(t *testing.T) {
	// Register a test module so IsModule returns true
	modules.Register(&modules.Module{Name: "visitortest"})

	// .value on a known module should not count as a task method
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.DotExpr{
				Object: &ast.IdentExpr{Name: "visitortest"},
				Field:  "value",
			}},
		},
	}
	if astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to ignore visitortest.value")
	}
}

func TestWalkExpressionsNestedSpawnInIf(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.IfStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Body: []ast.Statement{
					&ast.AssignStmt{Target: "t", Value: &ast.LoweredSpawnExpr{
						Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
					}},
				},
			},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn nested in if body")
	}
}

func TestWalkExpressionsNestedInFor(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ForStmt{
				Var:        "x",
				Collection: &ast.ArrayLiteral{Elements: []ast.Expr{&ast.IntLiteral{Value: "1"}}},
				Body: []ast.Statement{
					&ast.ExprStmt{Expression: &ast.LoweredParallelExpr{
						Branches: []ast.ParallelBranch{{Expr: &ast.IntLiteral{Value: "1"}, Index: 0}},
					}},
				},
			},
		},
	}
	if !astUsesParallel(prog) {
		t.Error("expected astUsesParallel to find parallel nested in for body")
	}
}

func TestWalkExpressionsNestedInFuncDef(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.FuncDef{
				Name: "foo",
				Body: []ast.Statement{
					&ast.ReturnStmt{Value: &ast.LoweredSpawnExpr{
						Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "42"}}},
					}},
				},
			},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn inside func def")
	}
}

func TestWalkExpressionsNestedInTry(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.LoweredTryExpr{
				Expr:   &ast.LoweredSpawnExpr{Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}}},
				ErrVar: "e",
				Handler: []ast.Statement{
					&ast.ExprStmt{Expression: &ast.NilLiteral{}},
				},
			}},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn inside try expr")
	}
}

func TestWalkExpressionsEmptyProgram(t *testing.T) {
	prog := &ast.Program{}
	if astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to be false on empty program")
	}
	if astUsesParallel(prog) {
		t.Error("expected astUsesParallel to be false on empty program")
	}
	if astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to be false on empty program")
	}
}

func TestWalkExpressionsNestedInWhile(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.WhileStmt{
				Condition: &ast.BoolLiteral{Value: true},
				Body: []ast.Statement{
					&ast.ExprStmt{Expression: &ast.DotExpr{
						Object: &ast.IdentExpr{Name: "t"},
						Field:  "done",
					}},
				},
			},
		},
	}
	if !astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to find .done in while body")
	}
}

func TestWalkExpressionsInElsifClause(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.IfStmt{
				Condition: &ast.BoolLiteral{Value: false},
				Body:      []ast.Statement{},
				ElsifClauses: []ast.ElsifClause{
					{
						Condition: &ast.BoolLiteral{Value: true},
						Body: []ast.Statement{
							&ast.ExprStmt{Expression: &ast.LoweredSpawnExpr{
								Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
							}},
						},
					},
				},
			},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn in elsif body")
	}
}

func TestWalkExpressionsInElseBody(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.IfStmt{
				Condition: &ast.BoolLiteral{Value: false},
				Body:      []ast.Statement{},
				ElseBody: []ast.Statement{
					&ast.ExprStmt{Expression: &ast.LoweredParallelExpr{
						Branches: []ast.ParallelBranch{{Expr: &ast.IntLiteral{Value: "1"}, Index: 0}},
					}},
				},
			},
		},
	}
	if !astUsesParallel(prog) {
		t.Error("expected astUsesParallel to find parallel in else body")
	}
}

func TestWalkExpressionsInTestDef(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.TestDef{
				Name: "test spawn",
				Body: []ast.Statement{
					&ast.ExprStmt{Expression: &ast.LoweredSpawnExpr{
						Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
					}},
				},
			},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn in test def")
	}
}

func TestWalkExpressionsInIndexAssign(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.IndexAssignStmt{
				Object: &ast.IdentExpr{Name: "arr"},
				Index:  &ast.IntLiteral{Value: "0"},
				Value: &ast.LoweredSpawnExpr{
					Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
				},
			},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn in index assign value")
	}
}

func TestWalkExpressionsInBinaryExpr(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.BinaryExpr{
				Left:  &ast.IntLiteral{Value: "1"},
				Op:    "+",
				Right: &ast.DotExpr{Object: &ast.IdentExpr{Name: "t"}, Field: "wait"},
			}},
		},
	}
	if !astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to find .wait in binary expr")
	}
}

func TestWalkExpressionsInCallArgs(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.CallExpr{
				Func: &ast.IdentExpr{Name: "puts"},
				Args: []ast.Expr{
					&ast.DotExpr{Object: &ast.IdentExpr{Name: "t"}, Field: "value"},
				},
			}},
		},
	}
	if !astUsesTaskMethods(prog) {
		t.Error("expected astUsesTaskMethods to find .value in call args")
	}
}

func TestWalkExpressionsInArrayLiteral(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.ArrayLiteral{
				Elements: []ast.Expr{
					&ast.LoweredSpawnExpr{Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}}},
				},
			}},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn in array literal")
	}
}

func TestWalkExpressionsInHashLiteral(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.HashLiteral{
				Pairs: []ast.HashPair{
					{
						Key:   &ast.StringLiteral{Value: "key"},
						Value: &ast.LoweredSpawnExpr{Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}}},
					},
				},
			}},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn in hash literal value")
	}
}

func TestWalkExpressionsSpawnInsideParallel(t *testing.T) {
	prog := &ast.Program{
		Statements: []ast.Statement{
			&ast.ExprStmt{Expression: &ast.LoweredParallelExpr{
				Branches: []ast.ParallelBranch{{
					Expr: &ast.LoweredSpawnExpr{
						Body: []ast.Statement{&ast.ExprStmt{Expression: &ast.IntLiteral{Value: "1"}}},
					},
					Index: 0,
				}},
			}},
		},
	}
	if !astUsesSpawn(prog) {
		t.Error("expected astUsesSpawn to find spawn nested inside parallel")
	}
	if !astUsesParallel(prog) {
		t.Error("expected astUsesParallel to find parallel")
	}
}
