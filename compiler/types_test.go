package compiler

import (
	"math/bits"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Bitmask layout invariants. These tests lock the locked design decision
// for union types (bitmask representation): every concrete type is a single
// power of two, TypeUnknown is the zero value, TypeDynamic is the absorbing
// bit. Together this lets unifyTypes degenerate to bitwise OR (with the
// documented numeric-promotion special case) and lets `t == TypeInt` keep
// its original "exactly int" semantics.
func TestRugoTypeBitmaskInvariants(t *testing.T) {
	t.Run("TypeUnknown is zero", func(t *testing.T) {
		assert.Equal(t, RugoType(0), TypeUnknown)
	})

	concrete := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeNil, TypeArray, TypeHash, TypeDynamic,
	}

	t.Run("every concrete constant is a single bit", func(t *testing.T) {
		for _, c := range concrete {
			ones := bits.OnesCount16(uint16(c))
			assert.Equalf(t, 1, ones,
				"expected %s to have exactly one bit set, got %d", c, ones)
		}
	})

	t.Run("concrete constants are pairwise disjoint", func(t *testing.T) {
		for i, a := range concrete {
			for _, b := range concrete[i+1:] {
				assert.Zerof(t, a&b,
					"expected %s and %s to be disjoint, got overlap %x",
					a, b, uint16(a&b))
			}
		}
	})
}

// IsUnion reports whether a type carries more than one concrete-type bit
// (ignoring the absorbing TypeDynamic bit). This is how the mismatch
// checker decides to walk Members().
func TestRugoTypeIsUnion(t *testing.T) {
	tests := []struct {
		name string
		t    RugoType
		want bool
	}{
		{"unknown is not a union", TypeUnknown, false},
		{"single int is not a union", TypeInt, false},
		{"dynamic alone is not a union", TypeDynamic, false},
		{"int|string is a union", TypeInt | TypeString, true},
		{"int|float|string is a union", TypeInt | TypeFloat | TypeString, true},
		{"int|dynamic is not a union (dynamic absorbed)", TypeInt | TypeDynamic, false},
		{"int|float is a union", TypeInt | TypeFloat, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.t.IsUnion())
		})
	}
}

// Has tests bit-membership of a (possibly single-bit) type within another
// (possibly multi-bit) type. Has is the core primitive the compat predicate
// and codegen audit will rely on.
func TestRugoTypeHas(t *testing.T) {
	u := TypeInt | TypeString
	assert.True(t, u.Has(TypeInt))
	assert.True(t, u.Has(TypeString))
	assert.False(t, u.Has(TypeFloat))
	assert.False(t, u.Has(TypeBool))

	// Dynamic membership: absorbing bit set means Has(TypeDynamic) is true.
	d := TypeDynamic
	assert.True(t, d.Has(TypeDynamic))
	assert.False(t, d.Has(TypeInt))
}

// Members returns the single-bit decomposition of a (possibly union) type
// in a stable, declaration-order sequence. Used by the union-aware
// diagnostics in the mismatch checker.
func TestRugoTypeMembers(t *testing.T) {
	t.Run("single bit returns itself", func(t *testing.T) {
		assert.Equal(t, []RugoType{TypeInt}, TypeInt.Members())
		assert.Equal(t, []RugoType{TypeString}, TypeString.Members())
	})

	t.Run("union returns members in declaration order", func(t *testing.T) {
		// Declaration order: int, float, string, bool, nil, array, hash, dynamic.
		assert.Equal(t,
			[]RugoType{TypeInt, TypeString},
			(TypeInt | TypeString).Members())
		assert.Equal(t,
			[]RugoType{TypeInt, TypeFloat, TypeString},
			(TypeInt | TypeFloat | TypeString).Members())
		assert.Equal(t,
			[]RugoType{TypeString, TypeNil},
			(TypeString | TypeNil).Members())
	})

	t.Run("unknown returns empty", func(t *testing.T) {
		assert.Empty(t, TypeUnknown.Members())
	})

	t.Run("union with dynamic includes dynamic member last", func(t *testing.T) {
		// Dynamic in a union is unusual (it's absorbing), but the
		// decomposition is still well-defined for debug/inspection use.
		assert.Equal(t,
			[]RugoType{TypeInt, TypeDynamic},
			(TypeInt | TypeDynamic).Members())
	})
}

// NarrowGoType returns the narrow Go type string for codegen when the type
// is a single concrete non-dynamic bit; otherwise "". Unions, dynamic, and
// unknown all fall through to the interface{} path.
func TestRugoTypeNarrowGoType(t *testing.T) {
	tests := []struct {
		name string
		t    RugoType
		want string
	}{
		{"int -> int", TypeInt, "int"},
		{"float -> float64", TypeFloat, "float64"},
		{"string -> string", TypeString, "string"},
		{"bool -> bool", TypeBool, "bool"},
		{"nil -> empty", TypeNil, ""},
		{"array -> empty", TypeArray, ""},
		{"hash -> empty", TypeHash, ""},
		{"dynamic -> empty", TypeDynamic, ""},
		{"unknown -> empty", TypeUnknown, ""},
		{"union int|string -> empty", TypeInt | TypeString, ""},
		{"union int|float -> empty", TypeInt | TypeFloat, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.t.NarrowGoType())
		})
	}
}

// String() prints unions in declaration order, joined by '|', so diagnostics
// have a stable, readable shape ("Integer|String", not "String|Integer").
func TestRugoTypeStringUnion(t *testing.T) {
	tests := []struct {
		name string
		t    RugoType
		want string
	}{
		{"single Integer", TypeInt, "Integer"},
		{"single Any (dynamic)", TypeDynamic, "Any"},
		{"unknown", TypeUnknown, "Unknown"},
		{"Integer|String", TypeInt | TypeString, "Integer|String"},
		{"Integer|Float|String", TypeInt | TypeFloat | TypeString, "Integer|Float|String"},
		{"String|Nil", TypeString | TypeNil, "String|Nil"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.t.String())
		})
	}
}

// IsResolved: a union with at least one non-dynamic bit is resolved (we know
// SOMETHING about it). TypeUnknown is not resolved. TypeDynamic alone is not
// resolved either (it's the absorbing "give up" bit and behaves like the
// original TypeDynamic int constant did).
func TestRugoTypeIsResolvedWithUnions(t *testing.T) {
	assert.False(t, TypeUnknown.IsResolved())
	assert.False(t, TypeDynamic.IsResolved())
	assert.True(t, TypeInt.IsResolved())
	assert.True(t, (TypeInt | TypeString).IsResolved())
	// A union that contains dynamic is still resolved iff there is at
	// least one non-dynamic bit. Dynamic on its own is unresolved.
	assert.True(t, (TypeInt | TypeDynamic).IsResolved())
}

// IsTyped: only single-bit concrete numeric/string/bool types qualify for
// narrow Go codegen. Unions never do (they fall back to interface{}).
func TestRugoTypeIsTypedWithUnions(t *testing.T) {
	assert.True(t, TypeInt.IsTyped())
	assert.True(t, TypeFloat.IsTyped())
	assert.True(t, TypeString.IsTyped())
	assert.True(t, TypeBool.IsTyped())
	assert.False(t, TypeArray.IsTyped())
	assert.False(t, TypeHash.IsTyped())
	assert.False(t, TypeNil.IsTyped())
	assert.False(t, TypeDynamic.IsTyped())
	assert.False(t, TypeUnknown.IsTyped())
	// Unions never qualify, even when every member would individually.
	assert.False(t, (TypeInt | TypeFloat).IsTyped())
	assert.False(t, (TypeInt | TypeString).IsTyped())
}

// unifyTypes is the join operation used at branch/loop merge points. The
// locked semantics are:
//   - TypeUnknown is neutral: unifyTypes(t, TypeUnknown) == t.
//   - TypeDynamic is absorbing: unifyTypes(t, TypeDynamic) == TypeDynamic.
//   - int + float collapse to float (numeric promotion, preserves narrow
//     Go codegen for arithmetic).
//   - Every other distinct pair produces a union via bitwise OR.
func TestUnifyTypesUnionFormation(t *testing.T) {
	t.Run("same type returns itself", func(t *testing.T) {
		assert.Equal(t, TypeInt, unifyTypes(TypeInt, TypeInt))
		assert.Equal(t, TypeString, unifyTypes(TypeString, TypeString))
	})

	t.Run("string + int forms a union", func(t *testing.T) {
		assert.Equal(t, TypeString|TypeInt, unifyTypes(TypeString, TypeInt))
		assert.Equal(t, TypeString|TypeInt, unifyTypes(TypeInt, TypeString))
	})

	t.Run("bool + string forms a union", func(t *testing.T) {
		assert.Equal(t, TypeBool|TypeString, unifyTypes(TypeBool, TypeString))
	})

	t.Run("nil + int forms a union (loop-may-not-execute shape)", func(t *testing.T) {
		assert.Equal(t, TypeNil|TypeInt, unifyTypes(TypeNil, TypeInt))
	})

	t.Run("union + new type extends the union", func(t *testing.T) {
		u := TypeInt | TypeString
		assert.Equal(t, TypeInt|TypeString|TypeBool, unifyTypes(u, TypeBool))
	})

	t.Run("union + union merges members", func(t *testing.T) {
		a := TypeInt | TypeString
		b := TypeString | TypeBool
		assert.Equal(t, TypeInt|TypeString|TypeBool, unifyTypes(a, b))
	})
}

func TestUnifyTypesNumericPromotion(t *testing.T) {
	// Numeric promotion is the single non-OR special case. It preserves
	// narrow Go codegen for `1 + 2.0` style expressions, which would
	// otherwise widen to interface{}.
	assert.Equal(t, TypeFloat, unifyTypes(TypeInt, TypeFloat))
	assert.Equal(t, TypeFloat, unifyTypes(TypeFloat, TypeInt))

	// Promotion does NOT apply when one side is a union (string|int +
	// float MUST stay a union -- narrowing to float would erase the
	// string member).
	assert.Equal(t, TypeString|TypeInt|TypeFloat,
		unifyTypes(TypeString|TypeInt, TypeFloat))
}

func TestUnifyTypesUnknownIsNeutral(t *testing.T) {
	assert.Equal(t, TypeInt, unifyTypes(TypeUnknown, TypeInt))
	assert.Equal(t, TypeInt, unifyTypes(TypeInt, TypeUnknown))
	assert.Equal(t, TypeString|TypeInt, unifyTypes(TypeUnknown, TypeString|TypeInt))
	assert.Equal(t, TypeUnknown, unifyTypes(TypeUnknown, TypeUnknown))
}

func TestUnifyTypesDynamicIsAbsorbing(t *testing.T) {
	assert.Equal(t, TypeDynamic, unifyTypes(TypeDynamic, TypeInt))
	assert.Equal(t, TypeDynamic, unifyTypes(TypeInt, TypeDynamic))
	assert.Equal(t, TypeDynamic, unifyTypes(TypeDynamic, TypeString|TypeBool))
	assert.Equal(t, TypeDynamic, unifyTypes(TypeDynamic, TypeDynamic))
	// A union with TypeDynamic set is also treated as dynamic (the bit is
	// absorbing); explicitly test the form produced by passing
	// `TypeInt|TypeDynamic` through unifyTypes.
	assert.Equal(t, TypeDynamic, unifyTypes(TypeInt|TypeDynamic, TypeString))
}

// TestRugoTypeBitsArePowersOfTwo asserts the bitmask layout invariant:
// every single-bit type constant is a distinct power of two, and the
// constants are pairwise-disjoint. This protects against future
// additions that might accidentally reuse a bit.
func TestRugoTypeBitsArePowersOfTwo(t *testing.T) {
	bits := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeArray, TypeHash, TypeNil, TypeDynamic,
	}
	seen := map[RugoType]bool{}
	for _, b := range bits {
		assert.NotZero(t, b, "bit must be non-zero")
		assert.Equal(t, RugoType(0), b&(b-1),
			"%s (0x%x) must be a power of two", b, uint16(b))
		assert.False(t, seen[b], "duplicate bit value %d", b)
		seen[b] = true
	}
}

// TestRugoTypeMembersOrderStable asserts Members() returns members in
// declaration order, and that the order is stable across repeated
// calls. Stable iteration is important for deterministic codegen and
// error messages.
func TestRugoTypeMembersOrderStable(t *testing.T) {
	u := TypeString | TypeInt | TypeNil | TypeArray
	first := u.Members()
	for i := 0; i < 5; i++ {
		again := u.Members()
		assert.Equal(t, first, again, "Members() must be stable across calls")
	}
	// Declaration order: Int, Float, String, Bool, Nil, Array, Hash.
	expected := []RugoType{TypeInt, TypeString, TypeNil, TypeArray}
	assert.Equal(t, expected, first)
}

// TestUnifyTypesCommutative asserts unifyTypes(a, b) == unifyTypes(b, a)
// for every pair of resolved types.
func TestUnifyTypesCommutative(t *testing.T) {
	types := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeArray, TypeHash, TypeNil, TypeDynamic,
	}
	for _, a := range types {
		for _, b := range types {
			ab := unifyTypes(a, b)
			ba := unifyTypes(b, a)
			assert.Equal(t, ab, ba,
				"unifyTypes(%s, %s) != unifyTypes(%s, %s)", a, b, b, a)
		}
	}
}

// TestUnifyTypesIdempotent asserts unifyTypes(t, t) == t for every type
// including unions.
func TestUnifyTypesIdempotent(t *testing.T) {
	cases := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeArray, TypeHash, TypeNil, TypeDynamic,
		TypeInt | TypeString,
		TypeInt | TypeString | TypeNil,
		TypeArray | TypeHash,
	}
	for _, ty := range cases {
		out := unifyTypes(ty, ty)
		assert.Equal(t, ty, out,
			"unifyTypes(%s, %s) = %s, want %s", ty, ty, out, ty)
	}
}

// TestUnifyTypesAssociativeNoNumeric asserts (a ∪ b) ∪ c == a ∪ (b ∪ c)
// for triples that don't involve numeric promotion. Numeric promotion
// (int+float -> float) breaks pure associativity by design (it's a
// non-OR special case), so we restrict the property to non-numeric
// types.
func TestUnifyTypesAssociativeNoNumeric(t *testing.T) {
	types := []RugoType{
		TypeString, TypeBool, TypeArray, TypeHash, TypeNil,
	}
	for _, a := range types {
		for _, b := range types {
			for _, c := range types {
				left := unifyTypes(unifyTypes(a, b), c)
				right := unifyTypes(a, unifyTypes(b, c))
				assert.Equalf(t, left, right,
					"(%s ∪ %s) ∪ %s = %s, but %s ∪ (%s ∪ %s) = %s",
					a, b, c, left, a, b, c, right)
			}
		}
	}
}

// TestUnifyTypesUnknownNeutralForAll asserts TypeUnknown is the neutral
// element for unifyTypes against every other type.
func TestUnifyTypesUnknownNeutralForAll(t *testing.T) {
	types := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeArray, TypeHash, TypeNil, TypeDynamic,
		TypeInt | TypeString, TypeArray | TypeNil,
	}
	for _, ty := range types {
		assert.Equal(t, ty, unifyTypes(ty, TypeUnknown),
			"unifyTypes(%s, unknown) != %s", ty, ty)
		assert.Equal(t, ty, unifyTypes(TypeUnknown, ty),
			"unifyTypes(unknown, %s) != %s", ty, ty)
	}
}

// TestUnifyTypesDynamicAbsorbsAll asserts TypeDynamic absorbs every
// other type (including unions).
func TestUnifyTypesDynamicAbsorbsAll(t *testing.T) {
	types := []RugoType{
		TypeInt, TypeFloat, TypeString, TypeBool,
		TypeArray, TypeHash, TypeNil,
		TypeInt | TypeString, TypeArray | TypeNil | TypeHash,
	}
	for _, ty := range types {
		assert.Equal(t, TypeDynamic, unifyTypes(ty, TypeDynamic),
			"unifyTypes(%s, dynamic) != dynamic", ty)
		assert.Equal(t, TypeDynamic, unifyTypes(TypeDynamic, ty),
			"unifyTypes(dynamic, %s) != dynamic", ty)
	}
}
