package compiler

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrintGoFile_Minimal(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("hello")`}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "package main\n")
	assert.Contains(t, got, "func main() {\n")
	assert.Contains(t, got, `fmt.Println("hello")`)
	assert.Contains(t, got, "}\n")
}

func TestPrintGoFile_Imports(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Imports: []GoImport{
			{Path: "fmt"},
			{Path: "os/exec", Alias: "_"},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "import (\n")
	assert.Contains(t, got, "\t\"fmt\"\n")
	assert.Contains(t, got, "\t_ \"os/exec\"\n")
	assert.Contains(t, got, ")\n")
}

func TestPrintGoFile_VarDecl(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Decls: []GoDecl{
			GoVarDecl{Name: "x", Type: "interface{}"},
			GoVarDecl{Name: "y", Type: "int", Value: GoRawExpr{Code: "42"}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "var x interface{}\n")
	assert.Contains(t, got, "var y int = 42\n")
}

func TestPrintGoFile_FuncDecl(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Decls: []GoDecl{
			GoFuncDecl{
				Name:   "add",
				Params: []GoParam{{Name: "a", Type: "int"}, {Name: "b", Type: "int"}},
				Return: "int",
				Body: []GoStmt{
					GoReturnStmt{Value: GoRawExpr{Code: "a + b"}},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "func add(a int, b int) int {\n")
	assert.Contains(t, got, "\treturn a + b\n")
	assert.Contains(t, got, "}\n")
}

func TestPrintGoFile_IfElse(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoIfStmt{
				Cond: GoRawExpr{Code: "x > 0"},
				Body: []GoStmt{
					GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("positive")`}},
				},
				ElseIf: []GoElseIf{
					{
						Cond: GoRawExpr{Code: "x == 0"},
						Body: []GoStmt{
							GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("zero")`}},
						},
					},
				},
				Else: []GoStmt{
					GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("negative")`}},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tif x > 0 {\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"positive\")\n")
	assert.Contains(t, got, "\t} else if x == 0 {\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"zero\")\n")
	assert.Contains(t, got, "\t} else {\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"negative\")\n")
	assert.Contains(t, got, "\t}\n")
}

func TestPrintGoFile_ForLoop(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoForStmt{
				Init: "i := 0",
				Cond: "i < 10",
				Post: "i++",
				Body: []GoStmt{
					GoExprStmt{Expr: GoRawExpr{Code: "fmt.Println(i)"}},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tfor i := 0; i < 10; i++ {\n")
}

func TestPrintGoFile_WhileLoop(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoForStmt{
				Cond: "x > 0",
				Body: []GoStmt{GoBreakStmt{}},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tfor x > 0 {\n")
	assert.Contains(t, got, "\t\tbreak\n")
}

func TestPrintGoFile_ForRange(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoForRangeStmt{
				Key:        "i",
				Value:      "v",
				Collection: GoRawExpr{Code: "items"},
				Body: []GoStmt{
					GoExprStmt{Expr: GoRawExpr{Code: "fmt.Println(i, v)"}},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tfor i, v := range items {\n")
}

func TestPrintGoFile_Defer(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoDeferStmt{Body: []GoStmt{
				GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("cleanup")`}},
			}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tdefer func() {\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"cleanup\")\n")
	assert.Contains(t, got, "\t}()\n")
}

func TestPrintGoFile_GoStmt(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoGoStmt{Body: []GoStmt{
				GoExprStmt{Expr: GoRawExpr{Code: "work()"}},
			}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tgo func() {\n")
	assert.Contains(t, got, "\t\twork()\n")
	assert.Contains(t, got, "\t}()\n")
}

func TestPrintGoFile_Switch(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoSwitchStmt{
				Tag: GoRawExpr{Code: "x"},
				Cases: []GoCase{
					{Values: []GoExpr{GoRawExpr{Code: "1"}}, Body: []GoStmt{
						GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("one")`}},
					}},
				},
				Default: []GoStmt{
					GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("other")`}},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tswitch x {\n")
	assert.Contains(t, got, "\tcase 1:\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"one\")\n")
	assert.Contains(t, got, "\tdefault:\n")
	assert.Contains(t, got, "\t\tfmt.Println(\"other\")\n")
}

func TestPrintGoFile_LineDirective(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoLineDirective{File: "test.rugo", Line: 5},
			GoExprStmt{Expr: GoRawExpr{Code: `fmt.Println("hi")`}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "//line test.rugo:5\n")
}

func TestPrintGoFile_Assign(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoAssignStmt{Target: "x", Op: ":=", Value: GoRawExpr{Code: "42"}},
			GoAssignStmt{Target: "x", Op: "=", Value: GoRawExpr{Code: "x + 1"}},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tx := 42\n")
	assert.Contains(t, got, "\tx = x + 1\n")
}

func TestPrintGoFile_RawDecl(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Decls: []GoDecl{
			GoRawDecl{Code: "// --- Runtime ---\nfunc helper() {}\n"},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "// --- Runtime ---\nfunc helper() {}\n")
}

func TestPrintGoFile_RawStmt(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoRawStmt{Code: "x := 1\ny := 2"},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tx := 1\n")
	assert.Contains(t, got, "\ty := 2\n")
}

func TestPrintGoFile_IIFE(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoAssignStmt{
				Target: "result",
				Op:     ":=",
				Value: GoIIFEExpr{
					ReturnType: "int",
					Body: []GoStmt{
						GoAssignStmt{Target: "x", Op: ":=", Value: GoRawExpr{Code: "42"}},
					},
					Result: GoRawExpr{Code: "x"},
				},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "func() int {\n")
	assert.Contains(t, got, "x := 42\n")
	assert.Contains(t, got, "return x\n")
	assert.Contains(t, got, "}()")
}

func TestPrintGoFile_NestedIndent(t *testing.T) {
	// Verify proper nesting: func > if > for > stmt
	f := &GoFile{
		Package: "main",
		Decls: []GoDecl{
			GoFuncDecl{
				Name: "test",
				Body: []GoStmt{
					GoIfStmt{
						Cond: GoRawExpr{Code: "true"},
						Body: []GoStmt{
							GoForStmt{
								Cond: "i < 10",
								Body: []GoStmt{
									GoExprStmt{Expr: GoRawExpr{Code: "work()"}},
								},
							},
						},
					},
				},
			},
		},
	}
	got := PrintGoFile(f)
	lines := strings.Split(got, "\n")
	// Find the work() line and verify it has 3 tabs (func > if > for)
	for _, line := range lines {
		if strings.Contains(line, "work()") {
			assert.True(t, strings.HasPrefix(line, "\t\t\t"), "expected 3 tabs, got: %q", line)
		}
	}
}

func TestPrintGoFile_MultiAssign(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Init: []GoStmt{
			GoMultiAssignStmt{
				Targets: []string{"x", "y"},
				Op:      ":=",
				Value:   GoRawExpr{Code: "getValues()"},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\tx, y := getValues()\n")
}

func TestPrintGoFile_BareReturn(t *testing.T) {
	f := &GoFile{
		Package: "main",
		Decls: []GoDecl{
			GoFuncDecl{
				Name: "noop",
				Body: []GoStmt{GoReturnStmt{}},
			},
		},
	}
	got := PrintGoFile(f)
	assert.Contains(t, got, "\treturn\n")
}
