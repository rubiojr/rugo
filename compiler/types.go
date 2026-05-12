package compiler

import (
	"math/bits"
	"strings"

	"github.com/rubiojr/rugo/ast"
)

// RugoType is a bitmask of concrete-type bits. A single-bit value (e.g.
// TypeInt) is a concrete type; multi-bit values are unions formed by
// unifying distinct concretes across control-flow joins. TypeDynamic is the
// absorbing "give up" bit -- when set, the value behaves like the legacy
// dynamic sentinel regardless of which other bits are set.
//
// The bitmask layout is the locked union representation. Each concrete bit
// is a power of two, and pairwise disjoint, so:
//   - `t == TypeInt` keeps its original "exactly int" semantics (a union
//     would never equal a single-bit constant).
//   - Set union is bitwise OR, so unifyTypes degenerates to `a | b` (with
//     the documented int+float numeric-promotion special case).
//   - Membership is `t & X != 0`.
//   - The type is still a comparable map key.
type RugoType uint16

const (
	// TypeUnknown means inference hasn't resolved the type yet.
	TypeUnknown RugoType = 0
	// TypeInt is an integer type (Go int).
	TypeInt RugoType = 1 << 0
	// TypeFloat is a floating-point type (Go float64).
	TypeFloat RugoType = 1 << 1
	// TypeString is a string type.
	TypeString RugoType = 1 << 2
	// TypeBool is a boolean type.
	TypeBool RugoType = 1 << 3
	// TypeNil is the nil literal type.
	TypeNil RugoType = 1 << 4
	// TypeArray is []interface{} (element types not tracked).
	TypeArray RugoType = 1 << 5
	// TypeHash is map[interface{}]interface{}.
	TypeHash RugoType = 1 << 6
	// TypeDynamic is the absorbing "explicitly unresolvable" bit. Falls
	// back to interface{} in codegen.
	TypeDynamic RugoType = 1 << 7
)

// declaredOrder lists concrete bits in their declaration order. Members()
// and String() use this to produce stable, readable output for diagnostics
// ("int|string", not "string|int").
var declaredOrder = []RugoType{
	TypeInt, TypeFloat, TypeString, TypeBool,
	TypeNil, TypeArray, TypeHash, TypeDynamic,
}

func (t RugoType) String() string {
	if t == TypeUnknown {
		return "unknown"
	}
	if t == TypeDynamic {
		return "any"
	}
	parts := make([]string, 0, 4)
	for _, bit := range declaredOrder {
		if t&bit == 0 {
			continue
		}
		parts = append(parts, singleBitName(bit))
	}
	if len(parts) == 0 {
		return "?"
	}
	return strings.Join(parts, "|")
}

// singleBitName returns the user-facing name for a single-bit RugoType.
// Callers are responsible for ensuring t is a single bit (or TypeDynamic);
// multi-bit values must be formatted via String(). This is the
// diagnostic-side name, NOT the Go-codegen name (use GoType() for that).
func singleBitName(t RugoType) string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float"
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	case TypeNil:
		return "nil"
	case TypeArray:
		return "array"
	case TypeHash:
		return "hash"
	case TypeDynamic:
		return "any"
	}
	return "?"
}

// IsNumeric returns true for the single-bit numeric types int and float.
// Unions are NOT considered numeric even when all members are numeric --
// numeric promotion already collapses int+float into TypeFloat via
// unifyTypes, so a multi-bit value here means at least one non-numeric
// member is present.
func (t RugoType) IsNumeric() bool {
	return t == TypeInt || t == TypeFloat
}

// IsResolved returns true if the type carries at least one concrete bit
// that isn't the absorbing TypeDynamic. TypeUnknown (zero) and TypeDynamic
// alone are unresolved -- callers should treat them as "we don't know
// enough to flag a mismatch".
func (t RugoType) IsResolved() bool {
	if t == TypeUnknown {
		return false
	}
	// Strip the absorbing bit; if anything else remains, we have a
	// concrete (or union-of-concrete) type.
	return t & ^TypeDynamic != 0
}

// IsTyped returns true only for single-bit narrow-Go-codegen types. Unions
// always fall back to interface{} in codegen.
func (t RugoType) IsTyped() bool {
	return t == TypeInt || t == TypeFloat || t == TypeString || t == TypeBool
}

// Has reports whether every bit of other is set in t. For single-bit
// queries (Has(TypeInt)) this is membership; for multi-bit queries
// (Has(TypeInt|TypeString)) it's subset.
func (t RugoType) Has(other RugoType) bool {
	if other == TypeUnknown {
		// Every type "has" the empty set; treat as true for symmetry.
		return true
	}
	return t&other == other
}

// IsUnion reports whether t carries more than one concrete-type bit,
// ignoring the absorbing TypeDynamic bit. A union with TypeDynamic set is
// not flagged as a union because TypeDynamic absorbs the rest at the
// compatibility layer.
func (t RugoType) IsUnion() bool {
	concrete := t & ^TypeDynamic
	return bits.OnesCount16(uint16(concrete)) > 1
}

// Members returns the single-bit decomposition of t in declaration order
// (int, float, string, bool, nil, array, hash, dynamic). Used by the
// mismatch checker to enumerate union members in diagnostics and by debug
// inspection.
func (t RugoType) Members() []RugoType {
	if t == TypeUnknown {
		return nil
	}
	out := make([]RugoType, 0, 4)
	for _, bit := range declaredOrder {
		if t&bit != 0 {
			out = append(out, bit)
		}
	}
	return out
}

// NarrowGoType returns the Go type string for narrow codegen, or "" when
// the value must fall back to interface{}. Unions, dynamic, unknown, and
// the compound types (array, hash, nil) all return "".
func (t RugoType) NarrowGoType() string {
	if !t.IsTyped() {
		return ""
	}
	return t.GoType()
}

// ParseTypeAnnotation converts a source-level type annotation (e.g. "int")
// into a RugoType. The second return value reports whether the name is a
// recognised type — callers should surface a user-facing error when false.
//
// Recognised names mirror what `type_of()` returns at runtime:
//
//	int, float, string, bool, array, hash, nil, any
//
// "any" is the explicit dynamic type — the inference engine treats it as
// "give up, leave as interface{}".
func ParseTypeAnnotation(name string) (RugoType, bool) {
	switch name {
	case "int":
		return TypeInt, true
	case "float":
		return TypeFloat, true
	case "string":
		return TypeString, true
	case "bool":
		return TypeBool, true
	case "array":
		return TypeArray, true
	case "hash":
		return TypeHash, true
	case "nil":
		return TypeNil, true
	case "any":
		return TypeDynamic, true
	}
	return TypeDynamic, false
}

// KnownTypeNames returns the set of valid type annotation names for use in
// error messages.
func KnownTypeNames() []string {
	return []string{"int", "float", "string", "bool", "array", "hash", "nil", "any"}
}

// GoType returns the Go type string for codegen, or "" for untyped.
func (t RugoType) GoType() string {
	switch t {
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float64"
	case TypeString:
		return "string"
	case TypeBool:
		return "bool"
	default:
		return ""
	}
}

// unifyTypes is the join operation used at branch / loop merge points (and
// historically at sequential reassignment; see infer.go for that policy).
//
// Semantics:
//   - TypeUnknown is neutral: unifyTypes(t, TypeUnknown) == t.
//   - TypeDynamic is absorbing: if either side has the TypeDynamic bit
//     set, the result is TypeDynamic.
//   - int + float collapse to TypeFloat (numeric promotion). This is the
//     single non-OR special case; it preserves narrow Go codegen for
//     arithmetic across mixed-numeric branches.
//   - Every other distinct pair produces a union via bitwise OR.
//
// Numeric promotion only applies when BOTH sides are single-bit numerics
// (TypeInt or TypeFloat). A union that happens to contain numeric members
// extends rather than collapses, because narrowing to float would erase
// the non-numeric members.
func unifyTypes(a, b RugoType) RugoType {
	if a == b {
		return a
	}
	if a == TypeUnknown {
		return b
	}
	if b == TypeUnknown {
		return a
	}
	if a.Has(TypeDynamic) || b.Has(TypeDynamic) {
		return TypeDynamic
	}
	// Numeric promotion: int+float -> float. Only when BOTH sides are
	// single-bit numerics; for unions we fall through to OR.
	if (a == TypeInt && b == TypeFloat) || (a == TypeFloat && b == TypeInt) {
		return TypeFloat
	}
	return a | b
}

// TypeInfo holds inferred type information for a program.
type TypeInfo struct {
	// ExprTypes maps expressions to their inferred types.
	ExprTypes map[ast.Expr]RugoType
	// FuncTypes maps function names to their inferred signatures.
	FuncTypes map[string]*FuncTypeInfo
	// VarTypes maps (scope, variable name) to their final inferred type.
	// Scope is the function name (or "" for top-level).
	VarTypes map[string]map[string]RugoType
	// VarUseTypes maps identifier read sites (IdentExpr nodes) to the
	// flow-sensitive type of the variable at that point in the program.
	// This differs from ExprTypes for an IdentExpr in that VarUseTypes
	// reflects the *current* type (replace semantics across reassignments
	// in straight-line code), whereas ExprTypes returns the conservative
	// storage union used by codegen for variable declarations.
	VarUseTypes map[ast.Expr]RugoType
}

// FuncTypeInfo holds the inferred signature for a function.
//
// AnnotatedArgs[i] reports whether ParamTypes[i] came from an explicit
// "name : Type" annotation (user-asserted) rather than inference. Annotated
// types are "sticky": the inferrer treats them as ground truth and reports a
// compile error if usage contradicts them.
//
// AnnotatedReturn reports the same for ReturnType.
type FuncTypeInfo struct {
	ParamTypes      []RugoType
	AnnotatedArgs   []bool
	ReturnType      RugoType
	AnnotatedReturn bool
	HasDefaults     bool // true if the function has params with default values (variadic signature)
}

// ExprType returns the inferred type of an expression, or TypeDynamic if unknown.
func (ti *TypeInfo) ExprType(e ast.Expr) RugoType {
	if t, ok := ti.ExprTypes[e]; ok {
		return t
	}
	return TypeDynamic
}

// VarType returns the inferred type of a variable in a given scope.
func (ti *TypeInfo) VarType(scope, name string) RugoType {
	if vars, ok := ti.VarTypes[scope]; ok {
		if t, ok := vars[name]; ok {
			return t
		}
	}
	return TypeDynamic
}
