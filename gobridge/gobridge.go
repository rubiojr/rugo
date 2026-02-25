// Package gobridge provides the Go standard library bridge registry for Rugo.
//
// Each whitelisted Go package has its own file (e.g. strings.go, math.go) that
// self-registers via init(). To add a new bridge, create a new file and call
// Register() — no other files need to change.
package gobridge

import (
	"fmt"
	"go/types"
	"sort"
	"strings"
)

// GoType represents a Go type that can be bridged to/from Rugo's interface{} system.
type GoType int

const (
	GoString GoType = iota
	GoInt
	GoFloat64
	GoBool
	GoByte
	GoStringSlice // []string
	GoByteSlice   // []byte — bridged as string in Rugo
	GoInt32       // int32 — bridged as int in Rugo
	GoInt64       // int64 — bridged as int in Rugo
	GoUint32      // uint32 — bridged as int in Rugo
	GoUint64      // uint64 — bridged as int in Rugo
	GoUint        // uint — bridged as int in Rugo
	GoUintptr     // uintptr — bridged as int in Rugo
	GoFloat32     // float32 — bridged as float in Rugo
	GoRune        // rune — bridged as first char of string in Rugo
	GoFunc        // func param — Rugo lambda adapted to typed Go func
	GoDuration    // time.Duration — bridged as int (milliseconds) in Rugo
	GoError       // error (panics on non-nil, invisible to Rugo)
	GoAny         // interface{}/any — passed through unchanged
)

// GoFuncType describes a Go function signature for lambda adapter generation.
type GoFuncType struct {
	Params           []GoType
	Returns          []GoType
	TypeCasts        map[int]string      // param index → named type cast (e.g., "qt6.ApplicationState")
	StructCasts      map[int]string      // param index → struct wrapper type (e.g., "rugo_struct_qt6_QListWidgetItem")
	StructParamValue map[int]bool        // param index → true if value type (not pointer), needs dereference
	FuncParamPointer map[int]bool        // param index → true if function param is *func(...)
	StructGoTypes    map[int]string      // param index → Go type string (e.g., "qt6.QDate", "qt6.QListWidgetItem")
	FuncTypes        map[int]*GoFuncType // param index → nested function signature (for func-in-func params)
}

// GoStructField describes a single field to extract from a Go struct return.
type GoStructField struct {
	GoField  string // Go field name (e.g., "Scheme") or method name (e.g., "Hostname")
	RugoKey  string // Rugo hash key (e.g., "scheme")
	Type     GoType // For return wrapping
	IsMethod bool   // true → call as _v.GoField()
	Expr     string // if set, raw Go expression using _v (overrides GoField/IsMethod)
}

// GoStructReturn describes how to decompose a Go struct return into a Rugo hash.
type GoStructReturn struct {
	Fields  []GoStructField
	Pointer bool // true if the return is a pointer (emit nil check)
}

// GoArrayType describes a fixed-size Go array for codegen (e.g., [32]byte).
// Size is known at registration time from the Go function signature.
type GoArrayType struct {
	Elem GoType // element type (e.g., GoByte)
	Size int    // compile-time array size (e.g., 32)
}

// ExternalTypeInfo describes an external named type discovered in function
// signatures during Go module introspection. These types come from the
// module's dependencies (not from the module itself) and are wrapped as
// opaque handles with DotGet/DotCall support.
type ExternalTypeInfo struct {
	PkgPath        string               // full import path (e.g., "github.com/mappu/miqt/qt6")
	PkgName        string               // Go package name (e.g., "qt6")
	GoName         string               // type name (e.g., "QWidget")
	Named          *types.Named         // resolved type info (nil if unresolved)
	Methods        []GoStructMethodInfo // bridgeable methods discovered on the type
	EmbeddedFields []GoStructFieldInfo  // embedded pointer-to-struct fields (with WrapType set)
}

// ExternalTypeKey returns the qualified key for an external type
// used in knownStructs and structWrappers maps.
func ExternalTypeKey(pkgPath, goName string) string {
	return pkgPath + "." + goName
}

// GoStructInfo describes a discovered Go struct type for wrapper generation.
// Used by the inspector to record struct metadata and by codegen to emit
// type-safe DotGet/DotSet wrapper structs.
type GoStructInfo struct {
	GoName   string               // PascalCase struct name (e.g., "Config")
	RugoName string               // snake_case name for constructor (e.g., "config")
	Fields   []GoStructFieldInfo  // exported fields with bridgeable types
	Methods  []GoStructMethodInfo // bridgeable methods
}

// GoStructFieldInfo describes a single exported field of a Go struct.
type GoStructFieldInfo struct {
	GoName    string // PascalCase field name (e.g., "Name")
	RugoName  string // snake_case field name (e.g., "name")
	Type      GoType // field type for conversion
	TypeCast  string // optional explicit cast for DotSet (e.g., "uint16", "os.FileMode")
	WrapType  string // if set, field is an opaque struct handle — wrap with this type
	WrapValue bool   // when WrapType is set, true means source field is a value (wrap as &field)
}

// GoStructMethodInfo describes a bridgeable method on a Go struct.
type GoStructMethodInfo struct {
	GoName            string              // PascalCase method name (e.g., "Add")
	RugoName          string              // snake_case method name (e.g., "add")
	Params            []GoType            // parameter types (excluding receiver)
	Returns           []GoType            // return types
	Variadic          bool                // last param is variadic
	StructCasts       map[int]string      // param index → wrapper type for struct handles
	StructParamValue  map[int]bool        // param index → true if value type (needs dereference)
	StructReturnWraps map[int]string      // return index → wrapper type for struct handles
	StructReturnValue map[int]bool        // return index → true if value type (not pointer)
	TypeCasts         map[int]string      // param index → named type cast (e.g., "qt6.GestureType")
	FuncParamPointer  map[int]bool        // param index → true if function param is *func(...)
	FuncTypes         map[int]*GoFuncType // param index → Go func signature for lambda adapters
}

// RuntimeHelper describes a Go helper function emitted into the generated code.
// Multiple functions can share a helper via the same Key — it will only be emitted once.
type RuntimeHelper struct {
	// Key is a unique identifier for dedup (e.g. "rugo_json_prepare").
	Key string
	// Code is the full Go source for the helper function(s), including trailing newline.
	Code string
}

// CodegenFunc is the signature for custom code generation callbacks.
// pkgBase is the resolved Go package name (respects aliases).
// args are the raw Go expressions for each argument.
// rugoName is the user-visible function name for error messages (e.g. "json.marshal").
type CodegenFunc func(pkgBase string, args []string, rugoName string) string

// GoFuncSig describes the signature of a Go stdlib function for bridging.
type GoFuncSig struct {
	// GoName is the PascalCase Go function name (e.g. "Contains").
	GoName string
	// Params are the parameter types in order.
	Params []GoType
	// Returns are the return types in order.
	// (T, error) → auto-panics on error, returns T.
	// (T, bool) → returns T if true, nil if false.
	Returns []GoType
	// FuncTypes maps param indices to their Go function signatures.
	// Only used when the corresponding Params[i] is GoFunc.
	FuncTypes map[int]*GoFuncType
	// FuncParamPointer marks GoFunc params declared as *func(...).
	// Codegen wraps adapted lambdas by taking an address of the typed func.
	FuncParamPointer map[int]bool
	// StructReturn describes how to decompose a struct return into a Rugo hash.
	// When set, the generic codegen uses this instead of the normal return handling.
	StructReturn *GoStructReturn
	// ArrayTypes maps return indices to fixed-size array metadata.
	// Codegen slices [N]T → []T then uses existing GoType handling.
	ArrayTypes map[int]*GoArrayType
	// Variadic indicates the last param is variadic.
	Variadic bool
	// TypeCasts maps param indices to Go named type casts (e.g., {1: "os.FileMode"}).
	// The codegen wraps the converted arg: os.FileMode(rugo_to_int(arg)).
	TypeCasts map[int]string
	// Doc is the documentation string shown by `rugo doc`.
	Doc string
	// Codegen, when set, overrides the default code generation for this function.
	// The bridge file owns its own codegen logic instead of codegen.go.
	Codegen CodegenFunc
	// RuntimeHelpers lists Go helper functions this bridge function needs.
	// Helpers are deduped by Key and emitted once into the generated code.
	RuntimeHelpers []RuntimeHelper
	// StructCasts maps param indices to wrapper type names for struct handle unwrapping.
	// Codegen emits: arg.(*WrapperType).v
	StructCasts map[int]string
	// StructParamValue marks struct params that are value types (not pointers).
	// Codegen dereferences wrapper values for these params: *arg.(*WrapperType).v
	StructParamValue map[int]bool
	// StructReturnWraps maps return indices to wrapper type names for struct handle wrapping.
	// Codegen emits: &WrapperType{v: returnVal}
	StructReturnWraps map[int]string
	// StructReturnValue marks struct returns that are value types (not pointers).
	// Codegen wraps these by taking an address of a local copy before storing in wrapper.
	StructReturnValue map[int]bool
}

// Package holds the registry of bridgeable functions for a Go package.
type Package struct {
	// Path is the full Go import path (e.g. "path/filepath").
	Path string
	// Funcs maps rugo_snake_case names to Go function signatures.
	Funcs map[string]GoFuncSig
	// Doc is the package-level documentation shown by `rugo doc`.
	Doc string
	// NoGoImport, when true, suppresses Go import emission for this package.
	// Used for packages implemented entirely via runtime helpers (e.g. slices, maps).
	NoGoImport bool
	// ExtraImports lists additional Go import paths needed by runtime helpers.
	// These are emitted alongside the package's own import (e.g. maps needs "sort").
	ExtraImports []string
	// External marks packages discovered via require (Go module introspection)
	// as opposed to statically registered bridge packages. External packages
	// need a blank reference (var _ = pkg.Func) to suppress unused import errors.
	External bool
	// Structs holds metadata about discovered Go struct types in external packages.
	// Used by codegen to emit wrapper types with DotGet/DotSet methods.
	Structs []GoStructInfo
}

// registry maps Go package paths to their bridge definitions.
var registry = map[string]*Package{}

// globalTypeWrappers maps fully-qualified Go type paths (e.g., "github.com/mappu/miqt/qt6.QWidget")
// to wrapper type names (e.g., "rugo_struct_qt6_QWidget"). This enables cross-package wrapper
// reuse: when package B discovers a type from package A as an external type, it reuses
// package A's existing wrapper instead of creating a duplicate rugo_ext_ wrapper.
var globalTypeWrappers = map[string]string{}

// stdlibPackages lists Go stdlib packages available for import.
// These are lazily introspected and registered on first access.
var stdlibPackages = []string{
	"crypto/md5",
	"crypto/sha256",
	"encoding/base64",
	"encoding/hex",
	"encoding/json",
	"html",
	"math",
	"math/rand/v2",
	"net/url",
	"os",
	"path",
	"path/filepath",
	"sort",
	"strconv",
	"strings",
	"time",
	"unicode",
}

// ensureStdlib lazily introspects and registers all well-known stdlib packages.
// Safe to call multiple times — packages already in the registry are skipped.
func ensureStdlib() {
	for _, path := range stdlibPackages {
		if _, ok := registry[path]; ok {
			continue
		}
		if pkg, err := InspectCompiledPackage(path); err == nil {
			registry[path] = pkg
		}
	}
}

// Register adds a Go package to the bridge registry.
func Register(pkg *Package) {
	registry[pkg.Path] = pkg
}

// RegisterTypeWrapper records that a fully-qualified Go type (e.g., "github.com/mappu/miqt/qt6.QWidget")
// has been wrapped as the given wrapper type name. Used for cross-package wrapper reuse.
func RegisterTypeWrapper(goTypePath, wrapperType string) {
	globalTypeWrappers[goTypePath] = wrapperType
}

// LookupTypeWrapper checks if a fully-qualified Go type already has a wrapper registered
// from another package. Returns the wrapper type name and true if found.
func LookupTypeWrapper(goTypePath string) (string, bool) {
	wt, ok := globalTypeWrappers[goTypePath]
	return wt, ok
}

// IsPackage returns true if the package is whitelisted for Go bridge.
func IsPackage(pkg string) bool {
	_, ok := registry[pkg]
	return ok
}

// PackageNames returns sorted names of all available Go bridge packages.
func PackageNames() []string {
	ensureStdlib()
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Lookup looks up a Go bridge function by package path and rugo name.
func Lookup(pkg, rugoName string) (*GoFuncSig, bool) {
	bp, ok := registry[pkg]
	if !ok {
		return nil, false
	}
	sig, ok := bp.Funcs[rugoName]
	if !ok {
		return nil, false
	}
	return &sig, true
}

// PackageForNS finds the Go package path given a namespace used in codegen.
// Checks aliases first (from goImports map), then falls back to default namespace.
func PackageForNS(ns string, goImports map[string]string) (string, bool) {
	for pkg, alias := range goImports {
		if alias == ns {
			return pkg, true
		}
	}
	for pkg := range goImports {
		if DefaultNS(pkg) == ns {
			return pkg, true
		}
	}
	return "", false
}

// DefaultNS returns the default namespace for a Go package path
// (last segment, e.g. "path/filepath" → "filepath").
// Handles versioned paths: "math/rand/v2" → "rand".
func DefaultNS(pkg string) string {
	parts := strings.Split(pkg, "/")
	last := parts[len(parts)-1]
	// Go versioned packages: the "v2", "v3" etc. suffix is not the package name
	if len(parts) >= 2 && len(last) >= 2 && last[0] == 'v' && last[1] >= '0' && last[1] <= '9' {
		return parts[len(parts)-2]
	}
	return last
}

// PackageFuncs returns the function registry for a package, or nil if not found.
// Used by codegen to scan for needed runtime helpers.
func PackageFuncs(pkg string) map[string]GoFuncSig {
	bp, ok := registry[pkg]
	if !ok {
		return nil
	}
	return bp.Funcs
}

// GetPackage returns the full Package definition for a given path, or nil.
// Lazily introspects stdlib packages if not yet registered, and falls back
// to compile-time introspection for arbitrary Go packages.
func GetPackage(pkg string) *Package {
	if p, ok := registry[pkg]; ok {
		return p
	}
	ensureStdlib()
	if p, ok := registry[pkg]; ok {
		return p
	}
	// Try on-demand introspection for any Go package (e.g., net/http).
	if p, err := InspectCompiledPackage(pkg); err == nil {
		registry[pkg] = p
		return p
	}
	return nil
}

// LookupByNS finds a package by its namespace (last path segment).
// Ensures well-known stdlib packages are registered before searching.
// Returns the package and true if found.
func LookupByNS(ns string) (*Package, bool) {
	for _, pkg := range registry {
		if DefaultNS(pkg.Path) == ns {
			return pkg, true
		}
	}
	// Try lazy introspection of well-known stdlib packages.
	ensureStdlib()
	for _, pkg := range registry {
		if DefaultNS(pkg.Path) == ns {
			return pkg, true
		}
	}
	return nil, false
}

// TypeConvToGo returns the Go expression to convert an interface{} arg to the given Go type.
func TypeConvToGo(argExpr string, t GoType) string {
	switch t {
	case GoString:
		return "rugo_to_string(" + argExpr + ")"
	case GoInt:
		return "rugo_to_int(" + argExpr + ")"
	case GoFloat64:
		return "rugo_to_float(" + argExpr + ")"
	case GoBool:
		return "rugo_to_bool(" + argExpr + ")"
	case GoByte:
		return "byte(rugo_to_int(" + argExpr + "))"
	case GoStringSlice:
		return "rugo_go_to_string_slice(" + argExpr + ")"
	case GoByteSlice:
		return "rugo_to_byte_slice(" + argExpr + ")"
	case GoInt32:
		return "int32(rugo_to_int(" + argExpr + "))"
	case GoInt64:
		return "int64(rugo_to_int(" + argExpr + "))"
	case GoUint32:
		return "uint32(rugo_to_int(" + argExpr + "))"
	case GoUint64:
		return "uint64(rugo_to_int(" + argExpr + "))"
	case GoUint:
		return "uint(rugo_to_int(" + argExpr + "))"
	case GoUintptr:
		return "uintptr(rugo_to_int(" + argExpr + "))"
	case GoFloat32:
		return "float32(rugo_to_float(" + argExpr + "))"
	case GoRune:
		return "rugo_first_rune(rugo_to_string(" + argExpr + "))"
	case GoDuration:
		return "time.Duration(rugo_to_int(" + argExpr + ")) * time.Millisecond"
	case GoError:
		return "func() error { _i := interface{}(" + argExpr + "); if _i == nil { return nil }; return _i.(error) }()"
	case GoAny:
		return "rugo_to_go(" + argExpr + ")"
	default:
		return argExpr
	}
}

func TypeCastTarget(cast string) string {
	return strings.TrimPrefix(cast, "assert:")
}

func ApplyTypeCast(expr, cast string) string {
	if strings.HasPrefix(cast, "assert:") {
		return fmt.Sprintf("%s.(%s)", expr, TypeCastTarget(cast))
	}
	if strings.HasPrefix(cast, "*") {
		return fmt.Sprintf("*%s(%s)", cast[1:], expr)
	}
	return fmt.Sprintf("%s(%s)", cast, expr)
}

// GoTypeGoName returns the raw Go type name for code generation.
func GoTypeGoName(t GoType) string {
	switch t {
	case GoString:
		return "string"
	case GoInt:
		return "int"
	case GoFloat64:
		return "float64"
	case GoBool:
		return "bool"
	case GoByte:
		return "byte"
	case GoInt32:
		return "int32"
	case GoInt64:
		return "int64"
	case GoUint32:
		return "uint32"
	case GoUint64:
		return "uint64"
	case GoUint:
		return "uint"
	case GoUintptr:
		return "uintptr"
	case GoFloat32:
		return "float32"
	case GoRune:
		return "rune"
	case GoStringSlice:
		return "[]string"
	case GoByteSlice:
		return "[]byte"
	case GoDuration:
		return "time.Duration"
	case GoError:
		return "error"
	default:
		return "interface{}"
	}
}

// SliceElemType returns the element GoType for a slice GoType.
// For example, GoStringSlice → GoString, GoByteSlice → GoByte.
// Returns GoAny and false for non-slice types.
func SliceElemType(t GoType) (GoType, bool) {
	switch t {
	case GoStringSlice:
		return GoString, true
	case GoByteSlice:
		return GoByte, true
	default:
		return GoAny, false
	}
}

// FuncAdapterConv generates a Go adapter that wraps a Rugo lambda into a typed Go function.
// argExpr is the Go expression for the Rugo lambda (interface{}).
// ft describes the target Go function signature.
func funcTypeName(ft *GoFuncType) string {
	// Build param types.
	var params []string
	for i, t := range ft.Params {
		// Struct-casted param: use the stored Go type name.
		if ft.StructCasts != nil {
			if _, ok := ft.StructCasts[i]; ok {
				goType := ""
				if ft.StructGoTypes != nil {
					goType = ft.StructGoTypes[i]
				}
				if goType == "" {
					goType = "interface{}"
				}
				// Value type: use as-is; pointer type: prepend *
				if ft.StructParamValue == nil || !ft.StructParamValue[i] {
					goType = "*" + goType
				}
				params = append(params, goType)
				continue
			}
		}
		if t == GoFunc {
			typeName := ""
			if ft.TypeCasts != nil {
				if cast, ok := ft.TypeCasts[i]; ok {
					typeName = TypeCastTarget(cast)
				}
			}
			if typeName == "" {
				typeName = "func(...interface{}) interface{}"
				if ft.FuncTypes != nil {
					if nested, ok := ft.FuncTypes[i]; ok && nested != nil {
						typeName = funcTypeName(nested)
					}
				}
			}
			if ft.FuncParamPointer != nil && ft.FuncParamPointer[i] {
				typeName = "*" + typeName
			}
			params = append(params, typeName)
			continue
		}
		typeName := GoTypeGoName(t)
		// Use named type if a cast is needed (e.g., qt6.ApplicationState instead of int).
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[i]; ok {
				typeName = TypeCastTarget(cast)
			}
		}
		params = append(params, typeName)
	}

	// Build return type.
	retType := ""
	if len(ft.Returns) == 1 {
		retType = " " + GoTypeGoName(ft.Returns[0])
		// Use named return type if a cast is needed.
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[-1]; ok {
				retType = " " + TypeCastTarget(cast)
			}
		}
	} else if len(ft.Returns) > 1 {
		var rets []string
		for i, t := range ft.Returns {
			typeName := GoTypeGoName(t)
			if ft.TypeCasts != nil {
				if cast, ok := ft.TypeCasts[-(i + 1)]; ok {
					typeName = TypeCastTarget(cast)
				}
			}
			rets = append(rets, typeName)
		}
		retType = " (" + strings.Join(rets, ", ") + ")"
	}

	return fmt.Sprintf("func(%s)%s", strings.Join(params, ", "), retType)
}

// FuncTypeName returns the Go type literal for a classified function signature.
func FuncTypeName(ft *GoFuncType) string {
	return funcTypeName(ft)
}

// FuncToLambdaConv wraps a typed Go function into a Rugo lambda-compatible callback.
// argExpr is the Go expression for the typed function value.
// ft describes the typed function signature.
func FuncToLambdaConv(argExpr string, ft *GoFuncType) string {
	// Build converted args for calling the typed Go function.
	var convArgs []string
	for i, p := range ft.Params {
		argExpr := fmt.Sprintf("_a[%d]", i)
		if ft.StructCasts != nil {
			if wrapType, ok := ft.StructCasts[i]; ok {
				expr := fmt.Sprintf("rugo_upcast_%s(%s).v", wrapType, argExpr)
				// Value-type struct params need dereference (*wrapper.v).
				if ft.StructParamValue != nil && ft.StructParamValue[i] {
					expr = fmt.Sprintf("*rugo_upcast_%s(%s).v", wrapType, argExpr)
				}
				convArgs = append(convArgs, expr)
				continue
			}
		}
		if p == GoFunc {
			isPtr := ft.FuncParamPointer != nil && ft.FuncParamPointer[i]
			source := argExpr
			if isPtr {
				source = "*" + source
			}
			if ft.FuncTypes != nil {
				if nested, ok := ft.FuncTypes[i]; ok && nested != nil {
					nestedConv := FuncAdapterConv(argExpr, nested)
					if isPtr {
						nestedConv = fmt.Sprintf("func() *%s { _f := %s; return &_f }()", funcTypeName(nested), nestedConv)
					}
					convArgs = append(convArgs, nestedConv)
					continue
				}
			}
			if isPtr {
				convArgs = append(convArgs, fmt.Sprintf("func() *func(...interface{}) interface{} { _f := %s.(func(...interface{}) interface{}); return &_f }()", source))
				continue
			}
			convArgs = append(convArgs, fmt.Sprintf("%s.(func(...interface{}) interface{})", source))
			continue
		}
		conv := TypeConvToGo(argExpr, p)
		// Apply named type cast if needed (e.g., qt6.GestureType(rugo_to_int(arg))).
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[i]; ok {
				conv = ApplyTypeCast(conv, cast)
			}
		}
		convArgs = append(convArgs, conv)
	}
	call := fmt.Sprintf("_f(%s)", strings.Join(convArgs, ", "))

	if len(ft.Returns) == 0 {
		return fmt.Sprintf("func(_a ...interface{}) interface{} { _f := %s; %s; return nil }", argExpr, call)
	}
	if len(ft.Returns) == 1 {
		return fmt.Sprintf("func(_a ...interface{}) interface{} { _f := %s; return %s }", argExpr, TypeWrapReturn(call, ft.Returns[0]))
	}

	// Multi-return: collect into []interface{}.
	var vars []string
	var elems []string
	for i, t := range ft.Returns {
		v := fmt.Sprintf("_v%d", i)
		vars = append(vars, v)
		elems = append(elems, TypeWrapReturn(v, t))
	}
	return fmt.Sprintf(
		"func(_a ...interface{}) interface{} { _f := %s; %s := %s; return []interface{}{%s} }",
		argExpr,
		strings.Join(vars, ", "),
		call,
		strings.Join(elems, ", "),
	)
}

func FuncAdapterConv(argExpr string, ft *GoFuncType) string {
	// Build param list: _p0 type0, _p1 type1, ...
	var params []string
	for i, t := range ft.Params {
		// Struct-casted param: use the stored Go type name.
		if ft.StructCasts != nil {
			if _, ok := ft.StructCasts[i]; ok {
				goType := ""
				if ft.StructGoTypes != nil {
					goType = ft.StructGoTypes[i]
				}
				if goType == "" {
					goType = "interface{}"
				}
				// Value type: use as-is; pointer type: prepend *
				if ft.StructParamValue == nil || !ft.StructParamValue[i] {
					goType = "*" + goType
				}
				params = append(params, fmt.Sprintf("_p%d %s", i, goType))
				continue
			}
		}
		if t == GoFunc {
			typeName := ""
			if ft.TypeCasts != nil {
				if cast, ok := ft.TypeCasts[i]; ok {
					typeName = TypeCastTarget(cast)
				}
			}
			if typeName == "" {
				typeName = "func(...interface{}) interface{}"
				if ft.FuncTypes != nil {
					if nested, ok := ft.FuncTypes[i]; ok && nested != nil {
						typeName = funcTypeName(nested)
					}
				}
			}
			if ft.FuncParamPointer != nil && ft.FuncParamPointer[i] {
				typeName = "*" + typeName
			}
			params = append(params, fmt.Sprintf("_p%d %s", i, typeName))
			continue
		}
		typeName := GoTypeGoName(t)
		// Use named type if a cast is needed (e.g., qt6.ApplicationState instead of int).
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[i]; ok {
				typeName = TypeCastTarget(cast)
			}
		}
		params = append(params, fmt.Sprintf("_p%d %s", i, typeName))
	}

	// Build return type
	retType := ""
	if len(ft.Returns) == 1 {
		retType = " " + GoTypeGoName(ft.Returns[0])
		// Use named return type if a cast is needed.
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[-1]; ok {
				retType = " " + TypeCastTarget(cast)
			}
		}
	}

	// Build args to pass to lambda: interface{}(wrap each param back to Rugo)
	var callArgs []string
	for i, t := range ft.Params {
		pName := fmt.Sprintf("_p%d", i)
		// Struct-casted param: wrap into Rugo opaque wrapper.
		if ft.StructCasts != nil {
			if wrapType, ok := ft.StructCasts[i]; ok {
				ref := pName
				// Value type: take address for the wrapper
				if ft.StructParamValue != nil && ft.StructParamValue[i] {
					ref = "&" + pName
				}
				callArgs = append(callArgs, fmt.Sprintf("interface{}(&%s{v: %s})", wrapType, ref))
				continue
			}
		}
		if t == GoFunc {
			source := pName
			if ft.FuncParamPointer != nil && ft.FuncParamPointer[i] {
				source = "*" + source
			}
			if ft.FuncTypes != nil {
				if nested, ok := ft.FuncTypes[i]; ok && nested != nil {
					callArgs = append(callArgs, "interface{}("+FuncToLambdaConv(source, nested)+")")
					continue
				}
			}
			callArgs = append(callArgs, "interface{}("+source+")")
			continue
		}
		switch t {
		case GoRune:
			callArgs = append(callArgs, "interface{}(string([]rune{"+pName+"}))")
		case GoByteSlice:
			callArgs = append(callArgs, "interface{}([]byte("+pName+"))")
		default:
			callArgs = append(callArgs, "interface{}("+pName+")")
		}
	}

	// Build return conversion
	retExpr := fmt.Sprintf("_fn(%s)", strings.Join(callArgs, ", "))
	if len(ft.Returns) == 1 {
		retExpr = TypeConvToGo(retExpr, ft.Returns[0])
		// Apply named return type cast if needed.
		if ft.TypeCasts != nil {
			if cast, ok := ft.TypeCasts[-1]; ok {
				retExpr = ApplyTypeCast(retExpr, cast)
			}
		}
	}

	// For void functions (no return values), don't use `return`.
	if len(ft.Returns) == 0 {
		return fmt.Sprintf("func(%s) { _fn := %s.(func(...interface{}) interface{}); %s }",
			strings.Join(params, ", "), argExpr, retExpr)
	}

	return fmt.Sprintf("func(%s)%s { _fn := %s.(func(...interface{}) interface{}); return %s }",
		strings.Join(params, ", "), retType, argExpr, retExpr)
}

// StructDecompCode generates the Go hash literal for struct decomposition.
// varName is the Go variable holding the struct (e.g., "_v").
func StructDecompCode(varName string, sr *GoStructReturn) string {
	var entries []string
	for _, f := range sr.Fields {
		var valExpr string
		if f.Expr != "" {
			valExpr = strings.ReplaceAll(f.Expr, "_v", varName)
		} else if f.IsMethod {
			valExpr = fmt.Sprintf("%s.%s()", varName, f.GoField)
		} else {
			valExpr = fmt.Sprintf("%s.%s", varName, f.GoField)
		}
		entries = append(entries, fmt.Sprintf("\t\t%q: %s,", f.RugoKey, TypeWrapReturn(valExpr, f.Type)))
	}
	return "map[interface{}]interface{}{\n" + strings.Join(entries, "\n") + "\n\t}"
}

// TypeWrapReturn returns the Go expression to wrap a Go return value to interface{}.
func TypeWrapReturn(expr string, t GoType) string {
	switch t {
	case GoInt:
		return "interface{}(int(" + expr + "))"
	case GoStringSlice:
		return "rugo_go_from_string_slice(" + expr + ")"
	case GoByteSlice:
		return "interface{}([]byte(" + expr + "))"
	case GoInt32:
		return "interface{}(int(" + expr + "))"
	case GoInt64:
		return "interface{}(int(" + expr + "))"
	case GoUint32:
		return "interface{}(int(" + expr + "))"
	case GoUint64:
		return "interface{}(int(" + expr + "))"
	case GoUint:
		return "interface{}(int(" + expr + "))"
	case GoUintptr:
		return "interface{}(int(" + expr + "))"
	case GoFloat32:
		return "interface{}(float64(" + expr + "))"
	case GoRune:
		return "func() interface{} { _r := " + expr + "; if _r == 0 { return interface{}(\"\") }; return interface{}(string(_r)) }()"
	case GoDuration:
		return "interface{}(int(" + expr + " / time.Millisecond))"
	default:
		return "interface{}(" + expr + ")"
	}
}

// GoTypeName returns a human-readable name for a GoType.
func GoTypeName(t GoType) string {
	switch t {
	case GoString:
		return "string"
	case GoInt:
		return "int"
	case GoFloat64:
		return "float"
	case GoBool:
		return "bool"
	case GoByte:
		return "byte"
	case GoStringSlice:
		return "[]string"
	case GoByteSlice:
		return "[]byte"
	case GoInt32:
		return "int32"
	case GoInt64:
		return "int64"
	case GoUint32:
		return "uint32"
	case GoUint64:
		return "uint64"
	case GoUint:
		return "uint"
	case GoUintptr:
		return "uintptr"
	case GoFloat32:
		return "float32"
	case GoRune:
		return "rune"
	case GoFunc:
		return "func"
	case GoDuration:
		return "duration"
	case GoError:
		return "error"
	default:
		return "any"
	}
}

// PanicOnErr returns the Go code snippet to panic with a bridge error message.
// Used by Codegen callbacks: `if _err != nil { <panicOnErr> }`.
func PanicOnErr(rugoName string) string {
	return `panic(rugo_bridge_err("` + rugoName + `", _err))`
}

// RuneHelper is a shared runtime helper for GoRune params.
// Bridge files using GoRune should include this in RuntimeHelpers.
// RuneHelper is loaded from helpers/rune.go via embed.
var RuneHelper = helperFromFile("rugo_first_rune", runeHelperSrc)

// StringSliceHelper is loaded from helpers/string_slice.go via embed.
var StringSliceHelper = helperFromFile("rugo_go_to_string_slice", stringSliceHelperSrc)

// ByteSliceHelper is loaded from helpers/byte_slice.go via embed.
var ByteSliceHelper = helperFromFile("rugo_to_byte_slice", byteSliceHelperSrc)

// PackageNeedsHelper reports whether any function in a package uses the given GoType
// in params or returns, including inside GoFunc adapter signatures and struct methods.
func PackageNeedsHelper(pkg string, target GoType) bool {
	bp := GetPackage(pkg)
	if bp == nil {
		return false
	}
	for _, sig := range bp.Funcs {
		for _, p := range sig.Params {
			if p == target {
				return true
			}
		}
		for _, r := range sig.Returns {
			if r == target {
				return true
			}
		}
		for _, ft := range sig.FuncTypes {
			for _, p := range ft.Params {
				if p == target {
					return true
				}
			}
			for _, r := range ft.Returns {
				if r == target {
					return true
				}
			}
		}
	}
	// Check struct method params/returns (e.g., bytes.Buffer.Read takes []byte).
	for _, si := range bp.Structs {
		for _, m := range si.Methods {
			for _, p := range m.Params {
				if p == target {
					return true
				}
			}
			for _, r := range m.Returns {
				if r == target {
					return true
				}
			}
		}
	}
	return false
}

// PackageNeedsRuneHelper reports whether any function in a package uses GoRune params,
// including GoRune inside GoFunc adapter signatures.
func PackageNeedsRuneHelper(pkg string) bool {
	return PackageNeedsHelper(pkg, GoRune)
}

// AllRuntimeHelpers returns deduplicated runtime helpers from all functions in a package.
func AllRuntimeHelpers(pkg string) []RuntimeHelper {
	funcs := PackageFuncs(pkg)
	if funcs == nil {
		return nil
	}
	seen := map[string]bool{}
	var helpers []RuntimeHelper
	for _, sig := range funcs {
		for _, h := range sig.RuntimeHelpers {
			if !seen[h.Key] {
				seen[h.Key] = true
				helpers = append(helpers, h)
			}
		}
	}
	return helpers
}
