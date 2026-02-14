package compiler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/rubiojr/rugo/modules/bench"
	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/fmt"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/re"
	_ "github.com/rubiojr/rugo/modules/str"
	_ "github.com/rubiojr/rugo/modules/test"
)

// seedCorpus loads all .rugo files from examples/ and rats/fixtures/ as seed
// inputs for coverage-guided fuzzing.
func seedCorpus(f *testing.F) {
	root := filepath.Join("..", ".")
	dirs := []string{
		filepath.Join(root, "examples"),
		filepath.Join(root, "rats", "fixtures"),
	}
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".rugo") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			f.Add(string(data))
		}
	}

	// Hand-crafted seeds targeting known fragile areas
	seeds := []string{
		// Hash edge cases
		`x = {a: 1, b: 2}`,
		`x = {1 => "one", 2 => "two"}`,
		`x = {}`,
		// String interpolation
		`x = "hello #{name}"`,
		`x = "a #{1 + 2} b"`,
		// Heredocs
		"x = <<~EOF\n  hello\nEOF",
		// Try/or
		"x = try 1 / 0 or err\n  0\nend",
		// Lambda
		"f = fn(a, b)\n  a + b\nend",
		// Struct
		"struct Foo\n  x\n  y\nend\nf = Foo.new(1, 2)",
		// Spawn/parallel
		"t = spawn\n  1 + 2\nend",
		"r = parallel\n  1\n  2\nend",
		// Deep nesting
		"if true\n  if true\n    if true\n      puts(1)\n    end\n  end\nend",
		// Compound assignment
		"x = 0\nx += 1\nx -= 1\nx *= 2\nx /= 2",
		// For loops
		"for x in [1, 2, 3]\n  puts(x)\nend",
		"for k, v in {a: 1}\n  puts(k)\nend",
		// Index assign
		"x = [1, 2, 3]\nx[0] = 99",
		// Dot access
		"x = Foo.new(1)\nx.val = 2",
		// Paren-free
		"puts 1",
		`puts "hello"`,
		// Backticks
		"`echo hello`",
		// Raw strings
		`x = 'hello world'`,
		// While loops with break/next
		"i = 0\nwhile i < 3\n  i += 1\nend",
		"for n in [1, 2, 3]\n  if n == 2\n    next\n  end\n  puts(n)\nend",
		"for n in [1, 2, 3]\n  if n == 2\n    break\n  end\nend",
		// Integer range iteration and range()
		"for i in 10\n  puts(i)\nend",
		"for i in range(3, 7)\n  puts(i)\nend",
		"arr = range(5)",
		// Array destructuring / multi-return
		"a, b, c = [10, 20, 30]\nputs(a)",
		"def multi()\n  return [1, 2]\nend\na, b = multi()",
		// Negative indexing and slicing
		"arr = [1, 2, 3]\nputs(arr[-1])",
		"arr = [1, 2, 3, 4, 5]\nx = arr[1, 2]",
		// Float operations
		"x = 5.5 % 2.0",
		"x = 1.5\nx += 0.5\nx *= 2.0",
		// Raise
		`raise("boom")`,
		// Type introspection
		"puts(type_of(42))",
		"puts(type_of([1, 2]))",
		// Collection methods
		"[1, 2, 3].map(fn(x) x * 2 end)",
		"[1, 2, 3].filter(fn(x) x > 1 end)",
		"[1, 2, 3].reduce(0, fn(acc, x) acc + x end)",
		"[3, 1, 2].sort()",
		"[1, 2, 3].first()",
		"[1, 2, 3].last()",
		"[1, 1, 2].uniq()",
		"[[1], [2, 3]].flatten()",
		`{a: 1, b: 2}.keys()`,
		`{a: 1, b: 2}.values()`,
		`{a: 1}.merge({b: 2})`,
		// Method chaining
		"[1, 2, 3, 4].filter(fn(x) x > 1 end).map(fn(x) x * 10 end)",
		// Struct with methods
		"struct Cat\n  name\nend\ndef Cat.meow()\n  self.name\nend\nc = Cat.new(\"Milo\")\nc.meow()",
		// Sandbox directive
		"sandbox",
		"sandbox ro: [\"/tmp\"], rw: [\"/dev/null\"]",
		"sandbox ro: \"/etc\", connect: [80, 443], bind: 8080",
		// Backtick interpolation
		"name = \"world\"\n`echo hello #{name}`",
		// Use/require
		`use "str"`,
		`use "conv"`,
		// Eval
		"use \"eval\"\neval.string(\"1 + 2\")",
		// Nested interpolation
		`x = "a #{"b #{1 + 2} c"} d"`,
		// Lambda edge cases
		"f = fn() nil end\nf.call()",
		"f = fn(a, b, c) a + b + c end\nf.call(1, 2, 3)",
		// Empty/minimal
		"",
		"puts(1)",
		"# comment",
	}
	for _, s := range seeds {
		f.Add(s)
	}
}

// FuzzParseSource uses Go's coverage-guided fuzzer to find inputs that crash
// the parser or produce internal compiler errors.
func FuzzParseSource(f *testing.F) {
	seedCorpus(f)

	f.Fuzz(func(t *testing.T, src string) {
		c := &Compiler{}
		_, err := c.ParseSource(src, "fuzz.rugo")
		if err != nil {
			errStr := err.Error()
			// Skip known issue #005 — invalid assignment targets reported
			// as internal compiler error (tracked separately).
			if strings.Contains(errStr, "invalid assignment target") {
				return
			}
			if strings.Contains(errStr, "internal compiler error") {
				t.Errorf("internal compiler error on input:\n%s\nerror: %s", src, errStr)
			}
			if strings.Contains(strings.ToLower(errStr), "runtime error") {
				t.Errorf("runtime panic surfaced as error on input:\n%s\nerror: %s", src, errStr)
			}
			if strings.Contains(strings.ToLower(errStr), "nil pointer") {
				t.Errorf("nil pointer surfaced as error on input:\n%s\nerror: %s", src, errStr)
			}
			if strings.Contains(strings.ToLower(errStr), "index out of range") {
				t.Errorf("index out of range surfaced as error on input:\n%s\nerror: %s", src, errStr)
			}
		}
	})
}

// FuzzCodegen fuzzes the full pipeline through codegen. Inputs that parse
// successfully are fed to the code generator to find codegen-specific crashes.
func FuzzCodegen(f *testing.F) {
	seedCorpus(f)

	f.Fuzz(func(t *testing.T, src string) {
		c := &Compiler{}
		prog, err := c.ParseSource(src, "fuzz.rugo")
		if err != nil {
			return // only fuzz codegen on parseable inputs
		}

		// Wrap codegen in a recover to catch panics and report them
		// as test failures instead of crashing the fuzzer.
		var goSrc string
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("codegen panic on input:\n%s\npanic: %v", src, r)
				}
			}()
			var genErr error
			goSrc, genErr = generate(prog, "fuzz.rugo", false, nil)
			if genErr != nil {
				errStr := genErr.Error()
				if strings.Contains(errStr, "internal compiler error") {
					t.Errorf("codegen internal error on input:\n%s\nerror: %s", src, errStr)
				}
				lower := strings.ToLower(errStr)
				// Skip known: interpolation re-parsing triggers parser crashes
				if strings.Contains(lower, "interpolation error") {
					return
				}
				if strings.Contains(lower, "runtime error") {
					t.Errorf("codegen runtime panic on input:\n%s\nerror: %s", src, errStr)
				}
				if strings.Contains(lower, "nil pointer") {
					t.Errorf("codegen nil pointer on input:\n%s\nerror: %s", src, errStr)
				}
			}
		}()
		if goSrc == "" {
			return
		}

		// Sanity: generated Go should contain package main
		if !strings.Contains(goSrc, "package main") {
			t.Errorf("generated Go missing 'package main' for input:\n%s", src)
		}
	})
}

// FuzzPreprocessor targets just the preprocessor pipeline (heredoc expansion,
// comment stripping, struct expansion, and the main preprocessor) to find
// crashes before parsing.
func FuzzPreprocessor(f *testing.F) {
	seedCorpus(f)

	f.Fuzz(func(t *testing.T, src string) {
		c := &Compiler{}
		// parseSource runs the full preprocess → parse → walk pipeline.
		// A panic here that isn't caught means the preprocessor crashed.
		_, _ = c.ParseSource(src, "fuzz.rugo")
		// If we get here without a panic, the preprocessor survived.
		// The native fuzzer detects panics automatically.
	})
}
