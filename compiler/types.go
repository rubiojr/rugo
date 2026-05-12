package compiler

import (
	"github.com/rubiojr/rugo/ast"
)

type RugoType int

const (
	// TypeUnknown means inference hasn't resolved the type yet.
	TypeUnknown RugoType = iota
	// TypeInt is an integer type (Go int).
	TypeInt
	// TypeFloat is a floating-point type (Go float64).
	TypeFloat
	// TypeString is a string type.
	TypeString
	// TypeBool is a boolean type.
	TypeBool
	// TypeNil is the nil literal type.
	TypeNil
	// TypeArray is []interface{} (element types not tracked).
	TypeArray
	// TypeHash is map[interface{}]interface{}.
	TypeHash
	// TypeDynamic means the type is explicitly unresolvable (mixed types,
	// external calls, etc.). Falls back to interface{} in codegen.
	TypeDynamic
)

func (t RugoType) String() string {
	switch t {
	case TypeUnknown:
		return "unknown"
	case TypeInt:
		return "int"
	case TypeFloat:
		return "float64"
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
		return "dynamic"
	default:
		return "?"
	}
}

// IsNumeric returns true for int and float types.
func (t RugoType) IsNumeric() bool {
	return t == TypeInt || t == TypeFloat
}

// IsResolved returns true if the type is concrete (not unknown or dynamic).
func (t RugoType) IsResolved() bool {
	return t != TypeUnknown && t != TypeDynamic
}

// IsTyped returns true if the type can be used for typed codegen
// (resolved and not a compound type like array/hash).
func (t RugoType) IsTyped() bool {
	return t == TypeInt || t == TypeFloat || t == TypeString || t == TypeBool
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

// unifyTypes merges two types. If they agree, returns that type.
// If either is unknown, returns the other. If they conflict, returns dynamic.
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
	if a == TypeDynamic || b == TypeDynamic {
		return TypeDynamic
	}
	// int + float → float (numeric promotion)
	if a.IsNumeric() && b.IsNumeric() {
		return TypeFloat
	}
	return TypeDynamic
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
