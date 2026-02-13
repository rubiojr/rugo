// Package gobridge provides the Go standard library bridge registry for Rugo.
//
// Each whitelisted Go package has its own file (e.g. strings.go, math.go) that
// self-registers via init(). To add a new bridge, create a new file and call
// Register() — no other files need to change.
package gobridge

import (
	"fmt"
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
	GoFloat32     // float32 — bridged as float in Rugo
	GoRune        // rune — bridged as first char of string in Rugo
	GoFunc        // func param — Rugo lambda adapted to typed Go func
	GoDuration    // time.Duration — bridged as int (milliseconds) in Rugo
	GoError       // error (panics on non-nil, invisible to Rugo)
)

// GoFuncType describes a Go function signature for lambda adapter generation.
type GoFuncType struct {
	Params  []GoType
	Returns []GoType
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
}

// registry maps Go package paths to their bridge definitions.
var registry = map[string]*Package{}

// Register adds a Go package to the bridge registry.
// If Extend() was called first (init ordering), merges the extended funcs.
func Register(pkg *Package) {
	if existing, ok := registry[pkg.Path]; ok {
		// Merge extended functions into the full package
		for name, sig := range existing.Funcs {
			pkg.Funcs[name] = sig
		}
	}
	registry[pkg.Path] = pkg
}

// Extend merges additional functions into an already-registered package.
// If the package isn't registered yet (init order), it creates a placeholder
// that Register() will merge into later.
func Extend(path string, funcs map[string]GoFuncSig) {
	pkg, ok := registry[path]
	if !ok {
		// Package not yet registered — create a stub that Register will merge into
		registry[path] = &Package{Path: path, Funcs: funcs}
		return
	}
	for name, sig := range funcs {
		pkg.Funcs[name] = sig
	}
}

// IsPackage returns true if the package is whitelisted for Go bridge.
func IsPackage(pkg string) bool {
	_, ok := registry[pkg]
	return ok
}

// PackageNames returns sorted names of all whitelisted Go bridge packages.
func PackageNames() []string {
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
func GetPackage(pkg string) *Package {
	return registry[pkg]
}

// LookupByNS finds a package by its namespace (last path segment).
// Returns the package and true if found.
func LookupByNS(ns string) (*Package, bool) {
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
		return "[]byte(rugo_to_string(" + argExpr + "))"
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
	case GoFloat32:
		return "float32(rugo_to_float(" + argExpr + "))"
	case GoRune:
		return "rugo_first_rune(rugo_to_string(" + argExpr + "))"
	case GoDuration:
		return "time.Duration(rugo_to_int(" + argExpr + ")) * time.Millisecond"
	default:
		return argExpr
	}
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
	case GoFloat32:
		return "float32"
	case GoRune:
		return "rune"
	default:
		return "interface{}"
	}
}

// FuncAdapterConv generates a Go adapter that wraps a Rugo lambda into a typed Go function.
// argExpr is the Go expression for the Rugo lambda (interface{}).
// ft describes the target Go function signature.
func FuncAdapterConv(argExpr string, ft *GoFuncType) string {
	// Build param list: _p0 type0, _p1 type1, ...
	var params []string
	for i, t := range ft.Params {
		params = append(params, fmt.Sprintf("_p%d %s", i, GoTypeGoName(t)))
	}

	// Build return type
	retType := ""
	if len(ft.Returns) == 1 {
		retType = " " + GoTypeGoName(ft.Returns[0])
	}

	// Build args to pass to lambda: interface{}(wrap each param back to Rugo)
	var callArgs []string
	for i, t := range ft.Params {
		pName := fmt.Sprintf("_p%d", i)
		switch t {
		case GoRune:
			callArgs = append(callArgs, "interface{}(string([]rune{"+pName+"}))")
		case GoByteSlice:
			callArgs = append(callArgs, "interface{}(string("+pName+"))")
		default:
			callArgs = append(callArgs, "interface{}("+pName+")")
		}
	}

	// Build return conversion
	retExpr := fmt.Sprintf("_fn(%s)", strings.Join(callArgs, ", "))
	if len(ft.Returns) == 1 {
		retExpr = TypeConvToGo(retExpr, ft.Returns[0])
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
	case GoStringSlice:
		return "rugo_go_from_string_slice(" + expr + ")"
	case GoByteSlice:
		return "interface{}(string(" + expr + "))"
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

// PackageNeedsHelper reports whether any function in a package uses the given GoType
// in params or returns, including inside GoFunc adapter signatures.
func PackageNeedsHelper(pkg string, target GoType) bool {
	funcs := PackageFuncs(pkg)
	for _, sig := range funcs {
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
