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
type FuncTypeInfo struct {
	ParamTypes []RugoType
	ReturnType RugoType
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
