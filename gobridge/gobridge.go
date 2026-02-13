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
	GoInt64       // int64 — bridged as int in Rugo
	GoRune        // rune — bridged as first char of string in Rugo
	GoFunc        // func param — Rugo lambda adapted to typed Go func
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
	// Variadic indicates the last param is variadic.
	Variadic bool
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
}

// registry maps Go package paths to their bridge definitions.
var registry = map[string]*Package{}

// Register adds a Go package to the bridge registry.
// Called from init() in each mapping file.
func Register(pkg *Package) {
	registry[pkg.Path] = pkg
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
	case GoInt64:
		return "int64(rugo_to_int(" + argExpr + "))"
	case GoRune:
		return "rugo_first_rune(rugo_to_string(" + argExpr + "))"
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
	case GoInt64:
		return "int64"
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
	case GoInt64:
		return "interface{}(int(" + expr + "))"
	case GoRune:
		return "func() interface{} { _r := " + expr + "; if _r == 0 { return interface{}(\"\") }; return interface{}(string(_r)) }()"
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
	case GoInt64:
		return "int64"
	case GoRune:
		return "rune"
	case GoFunc:
		return "func"
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
var RuneHelper = RuntimeHelper{
	Key: "rugo_first_rune",
	Code: `func rugo_first_rune(s string) rune {
	for _, r := range s { return r }
	return 0
}

`,
}

// PackageNeedsRuneHelper reports whether any function in a package uses GoRune params,
// including GoRune inside GoFunc adapter signatures.
func PackageNeedsRuneHelper(pkg string) bool {
	funcs := PackageFuncs(pkg)
	for _, sig := range funcs {
		for _, p := range sig.Params {
			if p == GoRune {
				return true
			}
		}
		for _, ft := range sig.FuncTypes {
			for _, p := range ft.Params {
				if p == GoRune {
					return true
				}
			}
			for _, r := range ft.Returns {
				if r == GoRune {
					return true
				}
			}
		}
	}
	return false
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
