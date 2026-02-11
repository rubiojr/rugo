// Package gobridge provides the Go standard library bridge registry for Rugo.
//
// Each whitelisted Go package has its own file (e.g. strings.go, math.go) that
// self-registers via init(). To add a new bridge, create a new file and call
// Register() — no other files need to change.
package gobridge

import (
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
	GoError       // error (panics on non-nil, invisible to Rugo)
)

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
	default:
		return argExpr
	}
}

// TypeWrapReturn returns the Go expression to wrap a Go return value to interface{}.
func TypeWrapReturn(expr string, t GoType) string {
	switch t {
	case GoStringSlice:
		return "rugo_go_from_string_slice(" + expr + ")"
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
