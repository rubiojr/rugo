package parser

import (
	"fmt"
	"strings"
	"testing"
)

func TestParseSimple(t *testing.T) {
	src := []byte("puts(\"hello\")\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(ast) == 0 {
		t.Fatal("Expected non-empty AST")
	}
	t.Logf("AST: %v", ast)
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseAssignment(t *testing.T) {
	src := []byte("x = 42\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("AST: %v", ast)
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseFunction(t *testing.T) {
	src := []byte("def greet(name)\nputs(name)\nend\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseIf(t *testing.T) {
	src := []byte("if x == 1\nputs(\"one\")\nelsif x == 2\nputs(\"two\")\nelse\nputs(\"other\")\nend\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseWhile(t *testing.T) {
	src := []byte("while x > 0\nx = x - 1\nend\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseArray(t *testing.T) {
	src := []byte("x = [1, 2, 3]\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseHash(t *testing.T) {
	src := []byte("x = {\"a\" => 1, \"b\" => 2}\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseStringLit(t *testing.T) {
	src := []byte("x = \"hello world\"\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseRequire(t *testing.T) {
	src := []byte("require \"helpers\"\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseReturn(t *testing.T) {
	src := []byte("def foo()\nreturn 42\nend\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseUnary(t *testing.T) {
	src := []byte("x = -1\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseBoolOps(t *testing.T) {
	src := []byte("x = a && b || c\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseNested(t *testing.T) {
	src := []byte("x = (1 + 2) * 3\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseMultipleStatements(t *testing.T) {
	src := []byte("x = 1\ny = 2\nputs(x + y)\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseFuncCall(t *testing.T) {
	src := []byte("foo(1, 2, 3)\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

func TestParseIndexAccess(t *testing.T) {
	src := []byte("x = arr[0]\n")
	p := &Parser{}
	ast, err := p.Parse("test.rg", src)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	t.Logf("Tree:\n%s", walkStr(p, ast, 0))
}

// walkStr walks the flat AST and returns a string representation
func walkStr(p *Parser, ast []int32, lvl int) string {
	var sb strings.Builder
	indent := strings.Repeat("  ", lvl)
	for len(ast) != 0 {
		next := int32(1)
		switch n := ast[0]; {
		case n < 0:
			sym := Symbol(-n)
			fmt.Fprintf(&sb, "%s%v\n", indent, sym)
			next = 2 + ast[1]
			sb.WriteString(walkStr(p, ast[2:next], lvl+1))
		default:
			tok := p.Token(n)
			fmt.Fprintf(&sb, "%s  tok: sep=%q src=%q [%v]\n", indent, tok.Sep(), tok.Src(), Symbol(tok.Ch))
		}
		ast = ast[next:]
	}
	return sb.String()
}
