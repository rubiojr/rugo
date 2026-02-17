package ast

// Transform rewrites an AST. Implementations must not mutate the input program.
type Transform interface {
	Name() string
	Transform(prog *Program) *Program
}

// TransformFunc adapts a named function to the Transform interface.
type TransformFunc struct {
	N string
	F func(*Program) *Program
}

func (t TransformFunc) Name() string                     { return t.N }
func (t TransformFunc) Transform(prog *Program) *Program { return t.F(prog) }

// Chain composes transforms left-to-right into a single Transform.
// Each transform receives the output of the previous one.
func Chain(transforms ...Transform) Transform {
	return TransformFunc{
		N: "chain",
		F: func(prog *Program) *Program {
			for _, t := range transforms {
				prog = t.Transform(prog)
			}
			return prog
		},
	}
}

// --- Copy-on-write traversal helpers ---
// These are used by transform passes to walk slices of AST nodes,
// only allocating a new slice when at least one element changes.

// mapSlice applies fn to each element. Returns (newSlice, true) if any
// element changed, or (original, false) if all elements are identical.
func mapSlice[T any](items []T, fn func(T) T) ([]T, bool) {
	var out []T
	modified := false
	for i, item := range items {
		newItem := fn(item)
		if any(newItem) != any(item) {
			if !modified {
				out = make([]T, len(items))
				copy(out[:i], items[:i])
				modified = true
			}
		}
		if modified {
			out[i] = newItem
		}
	}
	if !modified {
		return items, false
	}
	return out, true
}
