package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainEmpty(t *testing.T) {
	prog := &Program{SourceFile: "test.rugo"}
	result := Chain().Transform(prog)
	assert.Same(t, prog, result, "empty chain returns same program")
}

func TestChainSingle(t *testing.T) {
	called := false
	transform := TransformFunc{
		N: "test",
		F: func(prog *Program) *Program {
			called = true
			return &Program{SourceFile: "modified"}
		},
	}
	prog := &Program{SourceFile: "original"}
	result := Chain(transform).Transform(prog)
	assert.True(t, called, "transform was called")
	assert.Equal(t, "modified", result.SourceFile)
}

func TestChainOrdering(t *testing.T) {
	var order []string
	t1 := TransformFunc{
		N: "first",
		F: func(prog *Program) *Program {
			order = append(order, "first")
			return prog
		},
	}
	t2 := TransformFunc{
		N: "second",
		F: func(prog *Program) *Program {
			order = append(order, "second")
			return prog
		},
	}
	t3 := TransformFunc{
		N: "third",
		F: func(prog *Program) *Program {
			order = append(order, "third")
			return prog
		},
	}
	prog := &Program{}
	Chain(t1, t2, t3).Transform(prog)
	assert.Equal(t, []string{"first", "second", "third"}, order)
}

func TestChainPipeline(t *testing.T) {
	// Each transform appends to SourceFile to verify chaining
	appendTransform := func(name, suffix string) Transform {
		return TransformFunc{
			N: name,
			F: func(prog *Program) *Program {
				return &Program{SourceFile: prog.SourceFile + suffix}
			},
		}
	}
	prog := &Program{SourceFile: "start"}
	result := Chain(
		appendTransform("a", "+a"),
		appendTransform("b", "+b"),
	).Transform(prog)
	assert.Equal(t, "start+a+b", result.SourceFile)
}

func TestChainOfChains(t *testing.T) {
	appendTransform := func(name, suffix string) Transform {
		return TransformFunc{
			N: name,
			F: func(prog *Program) *Program {
				return &Program{SourceFile: prog.SourceFile + suffix}
			},
		}
	}
	inner := Chain(appendTransform("a", "+a"), appendTransform("b", "+b"))
	outer := Chain(inner, appendTransform("c", "+c"))
	prog := &Program{SourceFile: "start"}
	result := outer.Transform(prog)
	assert.Equal(t, "start+a+b+c", result.SourceFile)
}

func TestChainName(t *testing.T) {
	c := Chain()
	assert.Equal(t, "chain", c.Name())
}

func TestTransformFuncName(t *testing.T) {
	tf := TransformFunc{N: "my-transform", F: func(p *Program) *Program { return p }}
	assert.Equal(t, "my-transform", tf.Name())
}

func TestConcurrencyLoweringName(t *testing.T) {
	cl := ConcurrencyLowering()
	assert.Equal(t, "concurrency-lowering", cl.Name())
}
