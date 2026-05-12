package compiler

import (
	"testing"

	"github.com/rubiojr/rugo/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// findAssignWithTarget returns the first AssignStmt with the given Target
// at top level.
func findAssignWithTarget(prog *ast.Program, target string) *ast.AssignStmt {
	for _, s := range prog.Statements {
		if as, ok := s.(*ast.AssignStmt); ok && as.Target == target {
			return as
		}
	}
	return nil
}

func TestVarUseTypes_BasicReplaceSemantics(t *testing.T) {
	src := `x = 1
a = x
x = "hello"
b = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)
	require.NotNil(t, ti.VarUseTypes, "TypeInfo.VarUseTypes should be initialized")

	aAssign := findAssignWithTarget(prog, "a")
	bAssign := findAssignWithTarget(prog, "b")
	require.NotNil(t, aAssign)
	require.NotNil(t, bAssign)

	assert.Equal(t, TypeInt, ti.VarUseTypes[aAssign.Value],
		"x used after `x = 1`: expected TypeInt, got %s", ti.VarUseTypes[aAssign.Value])
	assert.Equal(t, TypeString, ti.VarUseTypes[bAssign.Value],
		"x used after `x = \"hello\"`: expected TypeString, got %s", ti.VarUseTypes[bAssign.Value])
}

func TestVarUseTypes_BranchMergeUnifies(t *testing.T) {
	src := `x = 1
if true
  x = "hello"
end
y = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)
	require.NotNil(t, ti.VarUseTypes)

	yAssign := findAssignWithTarget(prog, "y")
	require.NotNil(t, yAssign)

	t1 := ti.VarUseTypes[yAssign.Value]
	assert.True(t, t1.Has(TypeInt) && t1.Has(TypeString),
		"after if/no-else, x should be int|string at the use site, got %s", t1)
}

func TestVarUseTypes_InsideBranchSeesReassigned(t *testing.T) {
	src := `x = 1
if true
  x = "hello"
  y = x
end
`
	prog := parseProgram(t, src)
	ti := Infer(prog)
	require.NotNil(t, ti.VarUseTypes)

	// Locate the inner `y = x` AssignStmt within the if-body.
	var inner *ast.AssignStmt
	for _, s := range prog.Statements {
		if ifs, ok := s.(*ast.IfStmt); ok {
			for _, inB := range ifs.Body {
				if as, ok := inB.(*ast.AssignStmt); ok && as.Target == "y" {
					inner = as
					break
				}
			}
		}
	}
	require.NotNil(t, inner, "expected an inner `y = x` AssignStmt")

	assert.Equal(t, TypeString, ti.VarUseTypes[inner.Value],
		"inside the if-body after x reassigned to string, x's per-use type should be string, got %s",
		ti.VarUseTypes[inner.Value])
}

func TestVarUseTypes_AfterWhileLoopMaybeNotExecuted(t *testing.T) {
	src := `x = 1
while false
  x = "h"
end
y = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	yAssign := findAssignWithTarget(prog, "y")
	require.NotNil(t, yAssign)

	t1 := ti.VarUseTypes[yAssign.Value]
	assert.True(t, t1.Has(TypeInt) && t1.Has(TypeString),
		"after while-loop that may not execute, x should be int|string, got %s", t1)
}

func TestVarUseTypes_AfterForLoopMaybeNotExecuted(t *testing.T) {
	src := `x = 1
for item in [1, 2, 3]
  x = "h"
end
y = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	yAssign := findAssignWithTarget(prog, "y")
	require.NotNil(t, yAssign)

	t1 := ti.VarUseTypes[yAssign.Value]
	assert.True(t, t1.Has(TypeInt) && t1.Has(TypeString),
		"after for-loop that may not execute, x should be int|string, got %s", t1)
}

func TestVarUseTypes_AfterCaseStmtNoElse(t *testing.T) {
	src := `x = 1
status = "ok"
case status
of "a"
  x = "h"
end
y = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	yAssign := findAssignWithTarget(prog, "y")
	require.NotNil(t, yAssign)

	t1 := ti.VarUseTypes[yAssign.Value]
	assert.True(t, t1.Has(TypeInt) && t1.Has(TypeString),
		"after case-stmt without else, x should be int|string, got %s", t1)
}

func TestVarUseTypes_AfterCaseExprNoElse(t *testing.T) {
	src := `x = 1
status = "ok"
r = case status
of "a" -> 99
end
y = x
`
	prog := parseProgram(t, src)
	ti := Infer(prog)

	yAssign := findAssignWithTarget(prog, "y")
	require.NotNil(t, yAssign)

	// `case` expr without of-body reassignment shouldn't change x.
	t1 := ti.VarUseTypes[yAssign.Value]
	assert.Equal(t, TypeInt, t1,
		"case-expr that doesn't reassign x: x stays int, got %s", t1)
}
