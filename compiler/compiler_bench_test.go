package compiler

import (
	"testing"
)

// Benchmark the full compilation pipeline (parse + walk + codegen).
func BenchmarkCompileHelloWorld(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		c := &Compiler{}
		_, err := c.Compile("../examples/hello.rg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileFunctions(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		c := &Compiler{}
		_, err := c.Compile("../examples/functions.rg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileControlFlow(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		c := &Compiler{}
		_, err := c.Compile("../examples/control_flow.rg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileStringInterpolation(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		c := &Compiler{}
		_, err := c.Compile("../examples/string_interpolation.rg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompileArraysHashes(b *testing.B) {
	b.ResetTimer()
	for b.Loop() {
		c := &Compiler{}
		_, err := c.Compile("../examples/arrays_hashes.rg")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark preprocessing only.
func BenchmarkPreprocess(b *testing.B) {
	src := `
use "conv"
def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end
x = 10
puts "fib(#{x}) = #{conv.to_s(fib(x))}"
`
	funcs := scanFuncDefs(src)
	b.ResetTimer()
	for b.Loop() {
		_, _, err := preprocess(src, funcs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Benchmark code generation from an already-parsed AST.
func BenchmarkCodegen(b *testing.B) {
	c := &Compiler{}
	result, err := c.Compile("../examples/functions.rg")
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for b.Loop() {
		_, err := generate(result.Program, "functions.rg", false)
		if err != nil {
			b.Fatal(err)
		}
	}
}
