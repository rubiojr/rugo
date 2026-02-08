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
}

// Package holds the registry of bridgeable functions for a Go package.
type Package struct {
	// Path is the full Go import path (e.g. "path/filepath").
	Path string
	// Funcs maps rugo_snake_case names to Go function signatures.
	Funcs map[string]GoFuncSig
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
		parts := strings.Split(pkg, "/")
		if parts[len(parts)-1] == ns {
			return pkg, true
		}
	}
	return "", false
}

// DefaultNS returns the default namespace for a Go package path
// (last segment, e.g. "path/filepath" → "filepath").
func DefaultNS(pkg string) string {
	parts := strings.Split(pkg, "/")
	return parts[len(parts)-1]
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
