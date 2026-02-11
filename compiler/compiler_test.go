package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/str"
	"github.com/rubiojr/rugo/parser"
)

// Helper to compile rugo source to Go code.
func compileToGo(t *testing.T, src string) string {
	t.Helper()
	prog := parseAndWalk(t, src)
	goSrc, err := generate(prog, "test.rugo", false)
	if err != nil {
		t.Fatalf("Generate error: %v", err)
	}
	return goSrc
}

// --- Preprocessor Tests ---

func TestStripComments(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"no comment", `x = 1`, `x = 1`},
		{"line comment", "x = 1 # comment\n", "x = 1 \n"},
		{"full line comment", "# this is a comment\nx = 1\n", "\nx = 1\n"},
		{"comment in string", `x = "hello # world"`, `x = "hello # world"`},
		{"comment after string", "x = \"hello\" # comment\n", "x = \"hello\" \n"},
		{"multiple comments", "# first\nx = 1 # second\ny = 2\n", "\nx = 1 \ny = 2\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := stripComments(tt.input)
			if err != nil {
				t.Fatalf("stripComments(%q) unexpected error: %v", tt.input, err)
			}
			if result != tt.expect {
				t.Errorf("stripComments(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestExpandHeredocs(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{
			"no heredoc",
			"x = 1\n",
			"x = 1\n",
		},
		{
			"basic interpolating",
			"x = <<TEXT\nHello\nWorld\nTEXT\n",
			"x = \"Hello\\nWorld\"\n",
		},
		{
			"squiggly strips indent",
			"x = <<~TEXT\n    Hello\n      World\n    !\nTEXT\n",
			"x = \"Hello\\n  World\\n!\"\n",
		},
		{
			"raw heredoc",
			"x = <<'TEXT'\nHello #{name}\nTEXT\n",
			"x = ('Hello #{name}')\n",
		},
		{
			"raw squiggly",
			"x = <<~'CODE'\n    def foo\n    end\nCODE\n",
			"x = ('def foo' + \"\\n\" + 'end')\n",
		},
		{
			"empty heredoc",
			"x = <<TEXT\nTEXT\n",
			"x = \"\"\n",
		},
		{
			"indented closing delimiter",
			"x = <<TEXT\nHello\n  TEXT\n",
			"x = \"Hello\"\n",
		},
		{
			"escapes double quotes",
			"x = <<TEXT\nHe said \"hi\"\nTEXT\n",
			"x = \"He said \\\"hi\\\"\"\n",
		},
		{
			"escapes backslashes",
			"x = <<TEXT\nC:\\path\nTEXT\n",
			"x = \"C:\\\\path\"\n",
		},
		{
			"multiple heredocs",
			"a = <<A\nfoo\nA\nb = <<B\nbar\nB\n",
			"a = \"foo\"\nb = \"bar\"\n",
		},
		{
			"preserves surrounding code",
			"y = 1\nx = <<TEXT\nhi\nTEXT\nz = 2\n",
			"y = 1\nx = \"hi\"\nz = 2\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := expandHeredocs(tt.input)
			if err != nil {
				t.Fatalf("expandHeredocs error: %v", err)
			}
			if result != tt.expect {
				t.Errorf("expandHeredocs(%q)\n  got:  %q\n  want: %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestExpandHeredocsError(t *testing.T) {
	_, _, err := expandHeredocs("x = <<TEXT\nHello\n")
	if err == nil {
		t.Fatal("expected error for unterminated heredoc")
	}
	if !strings.Contains(err.Error(), "unterminated heredoc") {
		t.Errorf("expected 'unterminated heredoc' in error, got: %v", err)
	}
}

func TestHasInterpolation(t *testing.T) {
	tests := []struct {
		input  string
		expect bool
	}{
		{"hello world", false},
		{"hello #{name}", true},
		{"#{a} and #{b}", true},
		{"no interpolation #", false},
	}
	for _, tt := range tests {
		if hasInterpolation(tt.input) != tt.expect {
			t.Errorf("hasInterpolation(%q) = %v, want %v", tt.input, !tt.expect, tt.expect)
		}
	}
}

func TestProcessInterpolation(t *testing.T) {
	format, exprs := processInterpolation("Hello #{name}, age #{age}")
	if format != "Hello %v, age %v" {
		t.Errorf("format = %q, want %q", format, "Hello %v, age %v")
	}
	if len(exprs) != 2 || exprs[0] != "name" || exprs[1] != "age" {
		t.Errorf("exprs = %v, want [name, age]", exprs)
	}
}

// --- Code Generation Tests ---

func TestGenHelloWorld(t *testing.T) {
	src := compileToGo(t, `puts("Hello, World!")`)
	if !strings.Contains(src, `rugo_puts(`) {
		t.Errorf("expected puts call in output:\n%s", src)
	}
	if !strings.Contains(src, "func main()") {
		t.Error("expected main function")
	}
}

func TestGenAssignment(t *testing.T) {
	src := compileToGo(t, "x = 42")
	if !strings.Contains(src, "x :=") {
		t.Errorf("expected := declaration:\n%s", src)
	}
}

func TestGenFunction(t *testing.T) {
	src := compileToGo(t, "def greet(name)\nputs(name)\nend\ngreet(\"hi\")")
	if !strings.Contains(src, "func rugofn_greet(") {
		t.Errorf("expected function definition:\n%s", src)
	}
	if !strings.Contains(src, "rugofn_greet(") {
		t.Errorf("expected function call:\n%s", src)
	}
}

func TestGenIf(t *testing.T) {
	src := compileToGo(t, "if true\nputs(\"yes\")\nend")
	// With type inference, bool literals don't need rugo_to_bool
	if !strings.Contains(src, "if true") && !strings.Contains(src, "if rugo_to_bool(") {
		t.Errorf("expected if condition:\n%s", src)
	}
}

func TestGenWhile(t *testing.T) {
	src := compileToGo(t, "while true\nputs(\"loop\")\nend")
	// With type inference, bool literals don't need rugo_to_bool
	if !strings.Contains(src, "for true") && !strings.Contains(src, "for rugo_to_bool(") {
		t.Errorf("expected for loop:\n%s", src)
	}
}

func TestGenReturn(t *testing.T) {
	src := compileToGo(t, "def foo()\nreturn 42\nend")
	if !strings.Contains(src, "return") {
		t.Errorf("expected return statement:\n%s", src)
	}
}

func TestGenArithmetic(t *testing.T) {
	src := compileToGo(t, "x = 1 + 2")
	// With type inference, typed int arithmetic uses native ops
	if !strings.Contains(src, "(1 + 2)") && !strings.Contains(src, "rugo_add(") {
		t.Errorf("expected arithmetic expression:\n%s", src)
	}
}

func TestGenComparison(t *testing.T) {
	src := compileToGo(t, "x = 1 == 2")
	// With type inference, typed int comparison uses native ops
	if !strings.Contains(src, "(1 == 2)") && !strings.Contains(src, `rugo_eq(`) {
		t.Errorf("expected comparison expression:\n%s", src)
	}
}

func TestGenArray(t *testing.T) {
	src := compileToGo(t, "x = [1, 2, 3]")
	if !strings.Contains(src, "[]interface{}{") {
		t.Errorf("expected array literal:\n%s", src)
	}
}

func TestGenHash(t *testing.T) {
	src := compileToGo(t, `x = {"a" => 1}`)
	if !strings.Contains(src, "map[interface{}]interface{}{") {
		t.Errorf("expected map literal:\n%s", src)
	}
}

func TestGenStringInterpolation(t *testing.T) {
	src := compileToGo(t, `name = "World"`+"\n"+`puts("Hello, #{name}!")`)
	if !strings.Contains(src, "fmt.Sprintf(") {
		t.Errorf("expected fmt.Sprintf for interpolation:\n%s", src)
	}
}

func TestGenBuiltins(t *testing.T) {
	builtins := []struct {
		call   string
		expect string
	}{
		{`puts("hi")`, "rugo_puts("},
		{`print("hi")`, "rugo_print("},
		{`len(x)`, "rugo_len("},
		{`append(x, 1)`, "rugo_append("},
	}
	for _, tt := range builtins {
		t.Run(tt.call, func(t *testing.T) {
			src := compileToGo(t, tt.call)
			if !strings.Contains(src, tt.expect) {
				t.Errorf("expected %q in output:\n%s", tt.expect, src)
			}
		})
	}
}

func TestGenStdlibCalls(t *testing.T) {
	tests := []struct {
		call   string
		expect string
	}{
		{`use "os"` + "\n" + `os.exec("ls")`, "rugo_os_exec("},
		{`use "os"` + "\n" + `os.exit(0)`, "rugo_os_exit("},
		{`use "http"` + "\n" + `http.get("url")`, "rugo_http_get("},
		{`use "http"` + "\n" + `http.post("url", "body")`, "rugo_http_post("},
		{`use "conv"` + "\n" + `conv.to_i("42")`, "rugo_conv_to_i("},
		{`use "conv"` + "\n" + `conv.to_f("3.14")`, "rugo_conv_to_f("},
		{`use "conv"` + "\n" + `conv.to_s(42)`, "rugo_conv_to_s("},
	}
	for _, tt := range tests {
		t.Run(tt.call, func(t *testing.T) {
			src := compileToGo(t, tt.call)
			if !strings.Contains(src, tt.expect) {
				t.Errorf("expected %q in output:\n%s", tt.expect, src)
			}
		})
	}
}

func TestGenUnary(t *testing.T) {
	src := compileToGo(t, "x = -1")
	// With type inference, typed int negation uses native ops
	if !strings.Contains(src, "(-1)") && !strings.Contains(src, "rugo_negate(") {
		t.Errorf("expected negation:\n%s", src)
	}
}

func TestGenNot(t *testing.T) {
	src := compileToGo(t, "x = !true")
	// With type inference, typed bool negation uses native ops
	if !strings.Contains(src, "(!true)") && !strings.Contains(src, "rugo_not(") {
		t.Errorf("expected not:\n%s", src)
	}
}

// --- Compiler Integration Tests ---

func TestCompilerCompile(t *testing.T) {
	// Write a temp Rugo file
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.rugo")
	os.WriteFile(file, []byte("puts(\"hello\")\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(file)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if !strings.Contains(result.GoSource, "func main()") {
		t.Error("expected main function in output")
	}
}

func TestCompilerRequire(t *testing.T) {
	tmpDir := t.TempDir()

	// Write helper file
	helperFile := filepath.Join(tmpDir, "helpers.rugo")
	os.WriteFile(helperFile, []byte("def double(n)\nreturn n * 2\nend\n"), 0644)

	// Write main file — uses namespaced call
	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"helpers\"\nputs(helpers.double(21))\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if !strings.Contains(result.GoSource, "rugons_helpers_double(") {
		t.Errorf("expected namespaced double function in output:\n%s", result.GoSource)
	}
}

func TestCompilerDuplicateRequire(t *testing.T) {
	tmpDir := t.TempDir()

	helperFile := filepath.Join(tmpDir, "helpers.rugo")
	os.WriteFile(helperFile, []byte("def foo()\nreturn 1\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"helpers\"\nrequire \"helpers\"\nputs(helpers.foo())\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	// Should only have one definition of foo
	count := strings.Count(result.GoSource, "func rugons_helpers_foo(")
	if count != 1 {
		t.Errorf("expected 1 definition of rugons_helpers_foo, got %d", count)
	}
}

func TestCompilerComments(t *testing.T) {
	c := &Compiler{}
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.rugo")
	os.WriteFile(file, []byte("# this is a comment\nputs(\"hello\") # inline\n"), 0644)

	result, err := c.Compile(file)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if !strings.Contains(result.GoSource, "rugo_puts") {
		t.Error("expected puts in output")
	}
}

// --- Runtime Tests (via generated code inspection) ---

func TestRuntimeFunctions(t *testing.T) {
	// Base runtime (always present)
	src := compileToGo(t, `puts("test")`)
	expectedRuntime := []string{
		"func rugo_to_bool(",
		"func rugo_to_int(",
		"func rugo_to_float(",
		"func rugo_to_string(",
		"func rugo_add(",
		"func rugo_sub(",
		"func rugo_mul(",
		"func rugo_div(",
		"func rugo_mod(",
		"func rugo_negate(",
		"func rugo_not(",
		"func rugo_eq(",
		"func rugo_neq(",
		"func rugo_lt(",
		"func rugo_gt(",
		"func rugo_le(",
		"func rugo_ge(",
		"func rugo_puts(",
		"func rugo_print(",
		"func rugo_shell(",
		"func rugo_len(",
		"func rugo_append(",
	}
	for _, fn := range expectedRuntime {
		if !strings.Contains(src, fn) {
			t.Errorf("missing runtime function: %s", fn)
		}
	}

	// os module runtime (only when imported)
	osSrc := compileToGo(t, `use "os"`+"\n"+`os.exec("ls")`)
	for _, fn := range []string{"func rugo_os_exec(", "func rugo_os_exit("} {
		if !strings.Contains(osSrc, fn) {
			t.Errorf("missing os module function: %s", fn)
		}
	}

	// http module runtime (only when imported)
	httpSrc := compileToGo(t, `use "http"`+"\n"+`http.get("url")`)
	for _, fn := range []string{"func rugo_http_get(", "func rugo_http_post("} {
		if !strings.Contains(httpSrc, fn) {
			t.Errorf("missing http module function: %s", fn)
		}
	}

	// conv module runtime (only when imported)
	convSrc := compileToGo(t, `use "conv"`+"\n"+`conv.to_s(1)`)
	for _, fn := range []string{"func rugo_conv_to_i(", "func rugo_conv_to_f(", "func rugo_conv_to_s("} {
		if !strings.Contains(convSrc, fn) {
			t.Errorf("missing conv module function: %s", fn)
		}
	}

	// Verify modules are NOT emitted when not imported
	if strings.Contains(src, "func rugo_os_exec(") {
		t.Error("os module should not be emitted without import")
	}
	if strings.Contains(src, "func rugo_http_get(") {
		t.Error("http module should not be emitted without import")
	}
	if strings.Contains(src, "func rugo_conv_to_i(") {
		t.Error("conv module should not be emitted without import")
	}
}

// --- Complex Program Tests ---

func TestComplexProgram(t *testing.T) {
	src := `
# Fibonacci calculator
use "conv"

def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

i = 0
while i < 10
  puts(conv.to_s(fib(i)))
  i = i + 1
end
`
	goSrc := compileToGo(t, src)
	if !strings.Contains(goSrc, "func rugofn_fib(") {
		t.Error("expected fib function")
	}
	// With type inference, the while condition may use native ops
	if !strings.Contains(goSrc, "for (i < 10)") && !strings.Contains(goSrc, "for rugo_to_bool(") {
		t.Error("expected while loop")
	}
}

func TestAllOperators(t *testing.T) {
	// With type inference, typed ops use native Go operators.
	// Test with mixed types to ensure runtime helpers still work.
	operators := []struct {
		src    string
		expect string
	}{
		{"x = [1]\ny = x[0] + 2", "rugo_add"},
		{"x = [1]\ny = x[0] - 2", "rugo_sub"},
		{"x = [1]\ny = x[0] * 2", "rugo_mul"},
		{"x = [1]\ny = x[0] / 2", "rugo_div"},
		{"x = [1]\ny = x[0] % 2", "rugo_mod"},
		{"x = [1]\ny = -x[0]", "rugo_negate"},
		{"x = [1]\ny = !x[0]", "rugo_not"},
	}
	for _, tt := range operators {
		t.Run(tt.src, func(t *testing.T) {
			goSrc := compileToGo(t, tt.src)
			if !strings.Contains(goSrc, tt.expect) {
				t.Errorf("expected %q in output for %q:\n%s", tt.expect, tt.src, goSrc)
			}
		})
	}
}

func TestVariableScoping(t *testing.T) {
	src := compileToGo(t, "x = 1\nx = 2")
	// First should be :=, second should be =
	lines := strings.Split(src, "\n")
	declCount := 0
	assignCount := 0
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "x :=") {
			declCount++
		} else if strings.HasPrefix(trimmed, "x =") {
			assignCount++
		}
	}
	if declCount != 1 {
		t.Errorf("expected 1 declaration, got %d", declCount)
	}
	if assignCount != 1 {
		t.Errorf("expected 1 reassignment, got %d", assignCount)
	}
}

func TestElsifChain(t *testing.T) {
	src := compileToGo(t, "if x == 1\nputs(\"a\")\nelsif x == 2\nputs(\"b\")\nelsif x == 3\nputs(\"c\")\nelse\nputs(\"d\")\nend")
	if strings.Count(src, "} else if") != 2 {
		t.Errorf("expected 2 else-if chains")
	}
	if !strings.Contains(src, "} else {") {
		t.Error("expected else clause")
	}
}

// === Phase 2: Preprocessor Tests ===

func TestPreprocessParenFreeBuiltin(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{`puts "hello"`, `puts("hello")`},
		{`puts "a", "b"`, `puts("a", "b")`},
		{`print "no newline"`, `print("no newline")`},
	}
	for _, tt := range tests {
		result, _, _ := preprocess(tt.input, nil)
		if strings.TrimSpace(result) != tt.expect {
			t.Errorf("preprocess(%q) = %q, want %q", tt.input, strings.TrimSpace(result), tt.expect)
		}
	}
}

func TestPreprocessParenFreeUserFunc(t *testing.T) {
	// Functions must be defined before paren-free calls are recognized at top level
	src := "def greet(name)\nputs(name)\nend\ndef send(a, b)\nputs(a)\nend\ngreet \"World\"\nsend \"foo\", \"bar\""
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")
	// greet "World" is on line index 6 (after two def...end blocks)
	if got := strings.TrimSpace(lines[6]); got != `greet("World")` {
		t.Errorf("greet line = %q, want %q", got, `greet("World")`)
	}
	// send "foo", "bar" is on line index 7
	if got := strings.TrimSpace(lines[7]); got != `send("foo", "bar")` {
		t.Errorf("send line = %q, want %q", got, `send("foo", "bar")`)
	}
}

func TestPreprocessShellFallbackSimple(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{`ls`, `__shell__("ls")`},
		{`whoami`, `__shell__("whoami")`},
		{`date`, `__shell__("date")`},
		{`ls -la`, `__shell__("ls -la")`},
		{`uname -a`, `__shell__("uname -a")`},
		{`echo "hello world"`, `__shell__("echo \"hello world\"")`},
		{`grep -r "TODO" .`, `__shell__("grep -r \"TODO\" .")`},
	}
	for _, tt := range tests {
		result, _, _ := preprocess(tt.input, nil)
		if strings.TrimSpace(result) != tt.expect {
			t.Errorf("preprocess(%q) = %q, want %q", tt.input, strings.TrimSpace(result), tt.expect)
		}
	}
}

func TestPreprocessLeavesAssignments(t *testing.T) {
	tests := []string{
		`x = 42`,
		`name = "hello"`,
		`x = 1 + 2`,
	}
	for _, input := range tests {
		result, _, _ := preprocess(input, nil)
		if strings.TrimSpace(result) != input {
			t.Errorf("preprocess(%q) = %q, should be unchanged", input, strings.TrimSpace(result))
		}
	}
}

func TestPreprocessLeavesKeywords(t *testing.T) {
	tests := []string{
		`if x == 1`,
		`while true`,
		`def foo()`,
		`end`,
		`return 42`,
		`require "helpers"`,
	}
	for _, input := range tests {
		result, _, _ := preprocess(input, nil)
		if strings.TrimSpace(result) != input {
			t.Errorf("preprocess(%q) = %q, should be unchanged", input, strings.TrimSpace(result))
		}
	}
}

func TestPreprocessLeavesParenCalls(t *testing.T) {
	tests := []string{
		`puts("hello")`,
		`foo(1, 2)`,
		`x.bar()`,
	}
	for _, input := range tests {
		result, _, _ := preprocess(input, nil)
		if strings.TrimSpace(result) != input {
			t.Errorf("preprocess(%q) = %q, should be unchanged", input, strings.TrimSpace(result))
		}
	}
}

func TestPreprocessShellWithPipes(t *testing.T) {
	input := `ls | grep foo`
	result, _, _ := preprocess(input, nil)
	expect := `__shell__("ls | grep foo")`
	if strings.TrimSpace(result) != expect {
		t.Errorf("preprocess(%q) = %q, want %q", input, strings.TrimSpace(result), expect)
	}
}

func TestPreprocessPreservesIndent(t *testing.T) {
	input := `  ls -la`
	result, _, _ := preprocess(input, nil)
	if !strings.HasPrefix(result, "  ") {
		t.Errorf("expected preserved indent in %q", result)
	}
}

func TestScanFuncDefs(t *testing.T) {
	src := "def greet(name)\nputs(name)\nend\ndef add(a, b)\nreturn a + b\nend\n"
	funcs := scanFuncDefs(src)
	if !funcs["greet"] {
		t.Error("expected greet to be found")
	}
	if !funcs["add"] {
		t.Error("expected add to be found")
	}
	if funcs["puts"] {
		t.Error("puts should not be found as user func")
	}
}

func TestPreprocessPositionalResolution(t *testing.T) {
	// Before def: identifier should be shell fallback
	// After def: identifier should be paren-free function call
	src := "ping google.com\ndef ping(s)\nputs(s)\nend\nping google.com"
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")

	// Line 0: before def → shell
	if got := strings.TrimSpace(lines[0]); got != `__shell__("ping google.com")` {
		t.Errorf("before def: got %q, want shell fallback", got)
	}
	// Line 4: after def...end → paren-free function call (arg passed as-is)
	if got := strings.TrimSpace(lines[4]); got != `ping(google.com)` {
		t.Errorf("after def: got %q, want paren-free function call", got)
	}
}

func TestPreprocessForwardRefsInFuncBody(t *testing.T) {
	// Inside a function body, forward references to later-defined functions should work
	src := "def foo()\nbar()\nend\ndef bar()\nputs(\"hi\")\nend\nfoo()"
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")

	// Line 1 (inside foo): bar() already has parens, left alone
	if got := strings.TrimSpace(lines[1]); got != `bar()` {
		t.Errorf("forward ref: got %q, want %q", got, `bar()`)
	}
}

func TestPreprocessForwardRefParenFreeInBody(t *testing.T) {
	// Paren-free forward reference inside a function body
	src := "def foo()\nbar \"hello\"\nend\ndef bar(s)\nputs(s)\nend"
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")

	// Line 1 (inside foo): bar "hello" → bar("hello") using allFuncs
	if got := strings.TrimSpace(lines[1]); got != `bar("hello")` {
		t.Errorf("forward paren-free: got %q, want %q", got, `bar("hello")`)
	}
}

func TestPreprocessNestedBlocksInDef(t *testing.T) {
	// if/while inside def should not confuse block tracking
	src := "def foo()\nif true\nputs(\"yes\")\nend\nend\nfoo()"
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")

	// Line 5: foo() after def...end — should be left alone (has parens)
	if got := strings.TrimSpace(lines[5]); got != `foo()` {
		t.Errorf("after nested def: got %q, want %q", got, `foo()`)
	}
}

func TestPreprocessShellBeforeDefFuncAfter(t *testing.T) {
	// echo is a common shell command; after def, it should call the function
	src := "echo \"from shell\"\ndef echo(msg)\nputs(msg)\nend\necho \"from function\""
	allFuncs := scanFuncDefs(src)
	result, _, _ := preprocess(src, allFuncs)
	lines := strings.Split(result, "\n")

	// Line 0: shell
	if got := strings.TrimSpace(lines[0]); got != `__shell__("echo \"from shell\"")` {
		t.Errorf("before def: got %q, want shell fallback", got)
	}
	// Line 4: function call
	if got := strings.TrimSpace(lines[4]); got != `echo("from function")` {
		t.Errorf("after def: got %q, want function call", got)
	}
}

// === Phase 2: Namespace Tests ===

func TestGenDotCall(t *testing.T) {
	// Unknown ns.func() should compile to rugo_dot_call (runtime dispatch)
	prog := parseAndWalk(t, `ns.func(1, 2)`)
	_, err := generate(prog, "test.rugo", false)
	if err != nil {
		t.Errorf("unexpected error for dot call: %v", err)
	}
}

func TestGenShellBuiltin(t *testing.T) {
	src := compileToGo(t, `__shell__("ls -la")`)
	if !strings.Contains(src, "rugo_shell(") {
		t.Errorf("expected rugo_shell call:\n%s", src)
	}
}

func TestRuntimeShellFunction(t *testing.T) {
	src := compileToGo(t, `puts("test")`)
	if !strings.Contains(src, "func rugo_shell(") {
		t.Errorf("expected rugo_shell runtime function:\n%s", src)
	}
}

func TestCompilerNamespacedRequire(t *testing.T) {
	tmpDir := t.TempDir()

	helperFile := filepath.Join(tmpDir, "math_utils.rugo")
	os.WriteFile(helperFile, []byte("def add(a, b)\nreturn a + b\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"math_utils\"\nputs(math_utils.add(1, 2))\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if !strings.Contains(result.GoSource, "func rugons_math_utils_add(") {
		t.Errorf("expected namespaced function definition:\n%s", result.GoSource)
	}
	if !strings.Contains(result.GoSource, "rugons_math_utils_add(") {
		t.Errorf("expected namespaced function call:\n%s", result.GoSource)
	}
}

func TestCompilerAliasedRequire(t *testing.T) {
	tmpDir := t.TempDir()

	helperFile := filepath.Join(tmpDir, "long_module_name.rugo")
	os.WriteFile(helperFile, []byte("def foo()\nreturn 1\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"long_module_name\" as \"m\"\nputs(m.foo())\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	if err != nil {
		t.Fatalf("Compile error: %v", err)
	}
	if !strings.Contains(result.GoSource, "func rugons_m_foo(") {
		t.Errorf("expected aliased function definition:\n%s", result.GoSource)
	}
}

func TestPreprocessFullPipeline(t *testing.T) {
	// Test that a full program with mixed shell/rugo preprocesses correctly
	src := "x = 42\nputs \"hello\"\nls -la\nif x > 0\necho \"yes\"\nend\n"
	cleaned, err := stripComments(src)
	if err != nil {
		t.Fatalf("stripComments error: %v", err)
	}
	userFuncs := scanFuncDefs(cleaned)
	result, _, _ := preprocess(cleaned, userFuncs)
	lines := strings.Split(result, "\n")

	expectations := map[int]string{
		0: `x = 42`,
		1: `puts("hello")`,
		2: `__shell__("ls -la")`,
		3: `if x > 0`,
		4: `__shell__("echo \"yes\"")`,
		5: `end`,
	}
	for i, expect := range expectations {
		got := strings.TrimSpace(lines[i])
		if got != expect {
			t.Errorf("line %d: got %q, want %q", i, got, expect)
		}
	}
}

func TestPreprocessLeavesImport(t *testing.T) {
	tests := []string{
		`use "http"`,
		`use "os"`,
		`use "conv"`,
	}
	for _, input := range tests {
		result, _, _ := preprocess(input, nil)
		if strings.TrimSpace(result) != input {
			t.Errorf("preprocess(%q) = %q, should be unchanged", input, strings.TrimSpace(result))
		}
	}
}

func TestGenBuiltinsIncludesShell(t *testing.T) {
	builtins := []struct {
		call   string
		expect string
	}{
		{`__shell__("ls")`, "rugo_shell("},
	}
	for _, tt := range builtins {
		t.Run(tt.call, func(t *testing.T) {
			src := compileToGo(t, tt.call)
			if !strings.Contains(src, tt.expect) {
				t.Errorf("expected %q in output:\n%s", tt.expect, src)
			}
		})
	}
}

func TestGenConditionalImports(t *testing.T) {
	// Without http import, should not contain io or net/http
	src := compileToGo(t, `puts("hello")`)
	if strings.Contains(src, `"io"`) {
		t.Error("io should not be imported without http")
	}
	if strings.Contains(src, `"net/http"`) {
		t.Error("net/http should not be imported without http")
	}

	// With http import, should contain io and net/http
	httpSrc := compileToGo(t, `use "http"`+"\n"+`http.get("url")`)
	if !strings.Contains(httpSrc, `"io"`) {
		t.Error("io should be imported with http")
	}
	if !strings.Contains(httpSrc, `"net/http"`) {
		t.Error("net/http should be imported with http")
	}
}

// --- Try/Or Error Handling Tests ---

func TestTrySugarLevel1(t *testing.T) {
	// try EXPR → expands to try EXPR or _err nil end
	src := `result = try os.exec("fail")` + "\n" + `puts(result)`
	expanded, _ := expandTrySugar(`use "os"` + "\n" + src)
	if !strings.Contains(expanded, "or _err") {
		t.Errorf("level 1 sugar should expand to 'or _err':\n%s", expanded)
	}
}

func TestTrySugarLevel2(t *testing.T) {
	// try EXPR or DEFAULT → expands to try EXPR or _err DEFAULT end
	src := `result = try os.exec("fail") or "fallback"` + "\n"
	expanded, _ := expandTrySugar(src)
	if !strings.Contains(expanded, "or _err") || !strings.Contains(expanded, `"fallback"`) {
		t.Errorf("level 2 sugar should expand with default:\n%s", expanded)
	}
}

func TestTrySugarLevel3Passthrough(t *testing.T) {
	// Block form "try EXPR or ident\n...\nend" should NOT be expanded
	src := "result = try os.exec(\"fail\") or err\n  nil\nend\n"
	expanded, _ := expandTrySugar(src)
	if strings.Contains(expanded, "_err") {
		t.Errorf("block form should not be expanded:\n%s", expanded)
	}
}

func TestGenTryExpr(t *testing.T) {
	src := compileToGo(t, `use "os"`+"\n"+`result = try os.exec("ls") or err`+"\n"+`  "fallback"`+"\n"+`end`)
	if !strings.Contains(src, "defer func()") {
		t.Errorf("try should generate defer/recover:\n%s", src)
	}
	if !strings.Contains(src, "recover()") {
		t.Errorf("try should use recover():\n%s", src)
	}
	if !strings.Contains(src, "fmt.Sprint(e)") {
		t.Error("try should convert panic value to string")
	}
}

func TestGenTryExprSilent(t *testing.T) {
	// Level 1: try EXPR (preprocessor expands to block form)
	src := compileToGo(t, `use "os"`+"\n"+`x = try os.exec("ls") or _err`+"\n"+`nil`+"\n"+`end`)
	if !strings.Contains(src, "defer func()") {
		t.Error("level 1 try should generate defer/recover")
	}
}

func TestGenTryExprDefault(t *testing.T) {
	// Level 2: try EXPR or DEFAULT (preprocessor expands to block form)
	src := compileToGo(t, `use "os"`+"\n"+`x = try os.exec("ls") or _err`+"\n"+`"default"`+"\n"+`end`)
	if !strings.Contains(src, "defer func()") {
		t.Error("level 2 try should generate defer/recover")
	}
	if !strings.Contains(src, `"default"`) {
		t.Error("level 2 try should include default value")
	}
}

func TestTryInCondition(t *testing.T) {
	src := compileToGo(t, `use "os"`+"\n"+`if try os.exec("test -f /etc/hosts") or _err`+"\n"+`"false"`+"\n"+`end`+"\n"+`puts("exists")`+"\n"+`end`)
	if !strings.Contains(src, "defer func()") {
		t.Error("try in condition should generate defer/recover")
	}
}

// --- Compound Assignment Tests ---

func TestCompoundAssignSimple(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"plus-equals", "x += 1", "x = x + 1"},
		{"minus-equals", "x -= 3", "x = x - 3"},
		{"times-equals", "x *= 2", "x = x * 2"},
		{"div-equals", "x /= 4", "x = x / 4"},
		{"mod-equals", "x %= 3", "x = x % 3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandCompoundAssign(tt.input)
			if result != tt.expect {
				t.Errorf("expandCompoundAssign(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCompoundAssignPreservesIndent(t *testing.T) {
	result := expandCompoundAssign("  x += 1")
	if result != "  x = x + 1" {
		t.Errorf("expected indent preserved, got %q", result)
	}
}

func TestCompoundAssignWithIndex(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"array-index", `arr[0] += 1`, `arr[0] = arr[0] + 1`},
		{"hash-key", `h["key"] += 1`, `h["key"] = h["key"] + 1`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandCompoundAssign(tt.input)
			if result != tt.expect {
				t.Errorf("expandCompoundAssign(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCompoundAssignInString(t *testing.T) {
	input := `x = "a += b"`
	result := expandCompoundAssign(input)
	if result != input {
		t.Errorf("should not modify string contents, got %q", result)
	}
}

func TestCompoundAssignCompilesToGo(t *testing.T) {
	src := "x = 10\nx += 5\n"
	cleaned, err := stripComments(src)
	if err != nil {
		t.Fatalf("stripComments error: %v", err)
	}
	userFuncs := scanFuncDefs(cleaned)
	preprocessed, _, _ := preprocess(cleaned, userFuncs)
	if !strings.Contains(preprocessed, "x = x + 5") {
		t.Errorf("preprocessor should desugar +=, got:\n%s", preprocessed)
	}
}

// --- For..In Loop Tests ---

func TestParseForIn(t *testing.T) {
	src := "for x in arr\nputs(x)\nend\n"
	p := &parser.Parser{}
	_, err := p.Parse("test.rugo", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
}

func TestParseForInWithIndex(t *testing.T) {
	src := "for x, i in arr\nputs(x)\nend\n"
	p := &parser.Parser{}
	_, err := p.Parse("test.rugo", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
}

func TestGenForIn(t *testing.T) {
	src := compileToGo(t, "for x in arr\nputs(x)\nend\n")
	if !strings.Contains(src, "rugo_iterable_default") {
		t.Error("single-var for..in should generate rugo_iterable_default call")
	}
	if !strings.Contains(src, "rugo_for_item") {
		t.Error("single-var for..in should assign from rugo_for_item")
	}
}

func TestGenForInWithIndex(t *testing.T) {
	src := compileToGo(t, "for i, x in arr\nputs(x)\nend\n")
	if !strings.Contains(src, "rugo_for_kv.Val") {
		t.Error("for..in with index should assign second var from rugo_for_kv.Val")
	}
	if !strings.Contains(src, "rugo_for_kv.Key") {
		t.Error("for..in with index should assign first var from rugo_for_kv.Key")
	}
}

// --- Break/Next Tests ---

func TestParseBreak(t *testing.T) {
	src := "while true\nbreak\nend\n"
	p := &parser.Parser{}
	_, err := p.Parse("test.rugo", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
}

func TestParseNext(t *testing.T) {
	src := "while true\nnext\nend\n"
	p := &parser.Parser{}
	_, err := p.Parse("test.rugo", []byte(src))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
}

func TestGenBreak(t *testing.T) {
	src := compileToGo(t, "while true\nbreak\nend\n")
	if !strings.Contains(src, "break") {
		t.Error("break should generate Go break")
	}
}

func TestGenNext(t *testing.T) {
	src := compileToGo(t, "while true\nnext\nend\n")
	if !strings.Contains(src, "continue") {
		t.Error("next should generate Go continue")
	}
}

// --- Index Assignment Tests ---

func TestGenIndexAssign(t *testing.T) {
	src := compileToGo(t, `arr[0] = 42`+"\n")
	if !strings.Contains(src, "rugo_index_set") {
		t.Error("index assignment should generate rugo_index_set call")
	}
}

func TestGenIndexAssignHash(t *testing.T) {
	src := compileToGo(t, `h["key"] = "val"`+"\n")
	if !strings.Contains(src, "rugo_index_set") {
		t.Error("hash assignment should generate rugo_index_set call")
	}
}

// --- Module edge-case regression tests ---

// Bug 186d619: LookupFunc should only match imported modules, not all registered.
// When a user module is aliased to a stdlib name but that module is NOT imported,
// calls should resolve to the user module function, not the stdlib wrapper.
func TestRequireAliasStdlibNameWithoutImport(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "helpers.rugo"), []byte("def upper(s)\nreturn \"CUSTOM: \" + s\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"helpers\" as \"str\"\nputs(str.upper(\"hello\"))\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	require.NoError(t, err)
	// Should use the user module function, NOT the stdlib wrapper
	assert.Contains(t, result.GoSource, "rugons_str_upper(", "should call user module function")
	assert.NotContains(t, result.GoSource, "rugo_str_upper(", "should NOT call stdlib wrapper when str is not imported")
}

// Bug 65f41d8: When both import and require use the same namespace, and a function
// exists in both, the compiler should error rather than silently shadow.
func TestRequireAliasConflictsWithImport(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "helpers.rugo"), []byte("def upper(s)\nreturn \"CUSTOM\"\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("use \"str\"\nrequire \"helpers\" as \"str\"\nputs(str.upper(\"hello\"))\n"), 0644)

	c := &Compiler{}
	_, err := c.Compile(mainFile)
	require.Error(t, err, "should error when require alias conflicts with imported module")
	assert.Contains(t, err.Error(), "str", "error should mention the conflicting name")
}

// Bug 6013a24: Imports inside required files should be propagated.
func TestRequiredFileImports(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "helpers.rugo"), []byte("use \"conv\"\ndef double_str(n)\nreturn conv.to_s(n * 2)\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"helpers\"\nputs(helpers.double_str(21))\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	require.NoError(t, err)
	// The conv module runtime should be emitted because helpers.rugo imports it
	assert.Contains(t, result.GoSource, "rugo_conv_to_s(", "conv wrapper call should exist")
	// The conv module struct/runtime must also be emitted
	assert.Contains(t, result.GoSource, "type Conv struct", "conv runtime should be emitted from required file's import")
}

// Bug 51ab700: import/require inside function bodies should produce an error.
func TestImportInsideFuncBodyErrors(t *testing.T) {
	tmpDir := t.TempDir()

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("def foo()\nuse \"conv\"\nend\n"), 0644)

	c := &Compiler{}
	_, err := c.Compile(mainFile)
	require.Error(t, err, "use inside function body should produce an error")
	assert.Contains(t, err.Error(), "top level", "error should mention top-level requirement")
}

func TestRequireInsideFuncBodyErrors(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "helpers.rugo"), []byte("def foo()\nreturn 1\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("def bar()\nrequire \"helpers\"\nend\n"), 0644)

	c := &Compiler{}
	_, err := c.Compile(mainFile)
	require.Error(t, err, "require inside function body should produce an error")
	assert.Contains(t, err.Error(), "top level", "error should mention top-level requirement")
}

func TestImportInsideIfBodyErrors(t *testing.T) {
	tmpDir := t.TempDir()

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("if true\nuse \"conv\"\nend\n"), 0644)

	c := &Compiler{}
	_, err := c.Compile(mainFile)
	require.Error(t, err, "use inside if body should produce an error")
}

// Bug 59e1dc8: Two requires aliased to same namespace with conflicting function names.
func TestDuplicateNamespaceFunctionErrors(t *testing.T) {
	tmpDir := t.TempDir()

	os.WriteFile(filepath.Join(tmpDir, "a.rugo"), []byte("def foo()\nreturn \"a\"\nend\n"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.rugo"), []byte("def foo()\nreturn \"b\"\nend\n"), 0644)

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("require \"a\" as \"ns\"\nrequire \"b\" as \"ns\"\nputs(ns.foo())\n"), 0644)

	c := &Compiler{}
	_, err := c.Compile(mainFile)
	require.Error(t, err, "duplicate function in same namespace should error")
	assert.Contains(t, err.Error(), "foo", "error should mention the duplicate function name")
}

// Bug 6ee382f: Duplicate use statements are silently deduplicated.
func TestDuplicateImportDeduplicates(t *testing.T) {
	tmpDir := t.TempDir()

	mainFile := filepath.Join(tmpDir, "main.rugo")
	os.WriteFile(mainFile, []byte("use \"conv\"\nuse \"conv\"\nputs(conv.to_s(42))\n"), 0644)

	c := &Compiler{}
	result, err := c.Compile(mainFile)
	require.NoError(t, err, "duplicate use should be silently deduplicated")
	// Module runtime should appear exactly once
	count := strings.Count(result.GoSource, "type Conv struct")
	assert.Equal(t, 1, count, "conv runtime should be emitted exactly once")
}

func TestExpandHashColonSyntax(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"simple colon key", `{foo: "bar"}`, `{"foo" => "bar"}`},
		{"multiple colon keys", `{a: 1, b: 2}`, `{"a" => 1, "b" => 2}`},
		{"mixed colon and arrow", `{a: 1, "b" => 2}`, `{"a" => 1, "b" => 2}`},
		{"underscore key", `{my_key: "val"}`, `{"my_key" => "val"}`},
		{"arrow unchanged", `{"a" => 1}`, `{"a" => 1}`},
		{"no rewrite inside double string", `"foo: bar"`, `"foo: bar"`},
		{"no rewrite inside single string", `'foo: bar'`, `'foo: bar'`},
		{"ident colon no space", `foo:bar`, `foo:bar`},
		{"multiline hash", "{name: \"Alice\",\nage: 30}", "{\"name\" => \"Alice\",\n\"age\" => 30}"},
		{"keyword as key", `{if: true}`, `{"if" => true}`},
		{"tab after colon", "{foo:\t1}", "{\"foo\" =>\t1}"},
		{"nested hash", `{a: {b: 1}}`, `{"a" => {"b" => 1}}`},
		{"empty hash unchanged", `{}`, `{}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandHashColonSyntax(tt.input)
			if got != tt.expect {
				t.Errorf("expandHashColonSyntax(%q) =\n  %q\nwant:\n  %q", tt.input, got, tt.expect)
			}
		})
	}
}
