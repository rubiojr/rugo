package gobridge

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// InspectedPackage holds the results of introspecting a Go source package.
type InspectedPackage struct {
	// Package is the bridge package ready for registration.
	Package *Package
	// GoModulePath is the Go module path from go.mod (e.g. "github.com/user/rugo-slug").
	GoModulePath string
	// Skipped lists functions that were found but not bridgeable, with reasons.
	Skipped []ClassifiedFunc
	// KnownStructs maps Go struct type names to true for reclassification.
	KnownStructs map[string]bool
	// NamedTypes maps Go struct type names to their *types.Named for method discovery.
	NamedTypes map[string]*types.Named
}

// InspectSourcePackage introspects a Go source directory and returns a bridge
// package with all bridgeable exported functions classified. It reads go.mod
// for the module path and uses go/types for best-effort type checking.
func InspectSourcePackage(dir string) (*InspectedPackage, error) {
	absDir, _ := filepath.Abs(dir)

	// Find go.mod — may be in this dir or a parent (sub-package case).
	goModDir, found := FindGoModDir(absDir)
	if !found {
		return nil, fmt.Errorf("no go.mod found in %s or parent directories", dir)
	}

	modulePath, err := ReadGoModulePath(filepath.Join(goModDir, "go.mod"))
	if err != nil {
		return nil, fmt.Errorf("reading go.mod: %w", err)
	}

	// For sub-packages, append the relative path to the module path.
	if absDir != goModDir {
		rel, _ := filepath.Rel(goModDir, absDir)
		modulePath = modulePath + "/" + filepath.ToSlash(rel)
	}

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, goSourceFilter, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing Go source: %w", err)
	}

	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no Go packages found in %s", modulePath)
	}

	// Pick the non-test package.
	var files []*ast.File
	var pkgName string
	for name, pkg := range pkgs {
		if strings.HasSuffix(name, "_test") {
			continue
		}
		pkgName = name
		for _, f := range pkg.Files {
			files = append(files, f)
		}
		break
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no non-test Go files in %s", modulePath)
	}

	// Best-effort type checking — errors from unresolvable external imports
	// are ignored. Functions using those types will be correctly blocked.
	conf := types.Config{
		Importer: importer.Default(),
		Error:    func(error) {},
	}
	typePkg, _ := conf.Check(pkgName, fset, files, nil)
	if typePkg == nil {
		return nil, fmt.Errorf("type checking failed for %s", modulePath)
	}

	var allFuncs []ClassifiedFunc
	var structInfos []GoStructInfo
	knownStructs := make(map[string]bool)    // Go type name → true
	namedTypes := make(map[string]*types.Named) // Go type name → Named type (for method discovery)
	scope := typePkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		// Discover exported struct types.
		if tn, ok := obj.(*types.TypeName); ok && tn.Exported() {
			named, isNamed := tn.Type().(*types.Named)
			st, ok := tn.Type().Underlying().(*types.Struct)
			if !ok {
				continue
			}
			si := classifyStructFields(name, st)
			structInfos = append(structInfos, *si)
			knownStructs[name] = true
			if isNamed {
				namedTypes[name] = named
			}
			continue
		}

		fn, ok := obj.(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok || sig.Recv() != nil {
			continue
		}
		rugoName := ToSnakeCase(name)
		bf := ClassifyFunc(name, rugoName, sig)
		allFuncs = append(allFuncs, bf)
	}

	sort.Slice(allFuncs, func(i, j int) bool {
		if allFuncs[i].Tier != allFuncs[j].Tier {
			return allFuncs[i].Tier < allFuncs[j].Tier
		}
		return allFuncs[i].RugoName < allFuncs[j].RugoName
	})

	// Split into bridgeable and skipped.
	funcs := make(map[string]GoFuncSig)
	var skipped []ClassifiedFunc
	for _, f := range allFuncs {
		if f.Tier == TierAuto || f.Tier == TierCastable {
			sig := GoFuncSig{
				GoName:   f.GoName,
				Params:   f.Params,
				Returns:  f.Returns,
				Variadic: f.Variadic,
			}
			if len(f.FuncTypes) > 0 {
				sig.FuncTypes = f.FuncTypes
			}
			if len(f.ArrayTypes) > 0 {
				sig.ArrayTypes = f.ArrayTypes
			}
			funcs[f.RugoName] = sig
		} else {
			skipped = append(skipped, f)
		}
	}

	if len(funcs) == 0 && len(structInfos) == 0 {
		reasons := make([]string, 0, len(skipped))
		for _, f := range skipped {
			reasons = append(reasons, fmt.Sprintf("  %s: %s (%s)", f.GoName, f.Reason, f.Tier))
		}
		return nil, fmt.Errorf("no bridgeable functions found in %s\n%s", modulePath, strings.Join(reasons, "\n"))
	}

	pkg := &Package{
		Path:     modulePath,
		Funcs:    funcs,
		Doc:      fmt.Sprintf("Functions from Go module %s.", modulePath),
		External: true,
		Structs:  structInfos,
	}

	return &InspectedPackage{
		Package:       pkg,
		GoModulePath:  modulePath,
		Skipped:       skipped,
		KnownStructs:  knownStructs,
		NamedTypes:    namedTypes,
	}, nil
}

// IsGoModuleDir returns true if dir contains .go source files and is part of
// a Go module (has go.mod in dir or a parent directory).
func IsGoModuleDir(dir string) bool {
	if !hasGoFiles(dir) {
		return false
	}
	_, found := FindGoModDir(dir)
	return found
}

// IsGoPackageDir returns true if dir contains .go source files (may be a
// sub-package within a larger Go module).
func IsGoPackageDir(dir string) bool {
	return hasGoFiles(dir)
}

func hasGoFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// FindGoModDir walks up from dir to find the nearest go.mod.
// Returns the directory containing go.mod and true, or ("", false).
func FindGoModDir(dir string) (string, bool) {
	dir, _ = filepath.Abs(dir)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

// goSourceFilter filters directory entries to only .go files (no tests).
func goSourceFilter(info os.FileInfo) bool {
	return strings.HasSuffix(info.Name(), ".go") && !strings.HasSuffix(info.Name(), "_test.go")
}

// ReadGoModulePath reads the module path from a go.mod file.
func ReadGoModulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module ")), nil
		}
	}
	return "", fmt.Errorf("no module directive found in %s", path)
}

// classifyStructFields examines an exported struct and returns a GoStructInfo
// with all exported fields that have bridgeable types. Structs with no
// bridgeable fields are still returned as opaque handles (empty Fields slice).
func classifyStructFields(goName string, st *types.Struct) *GoStructInfo {
	var fields []GoStructFieldInfo
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() || f.Embedded() {
			continue
		}
		gt, tier, _ := ClassifyGoType(f.Type(), true)
		if tier == TierBlocked || tier == TierFunc {
			continue
		}
		fields = append(fields, GoStructFieldInfo{
			GoName:   f.Name(),
			RugoName: ToSnakeCase(f.Name()),
			Type:     gt,
		})
	}
	return &GoStructInfo{
		GoName:   goName,
		RugoName: ToSnakeCase(goName),
		Fields:   fields,
	}
}

// FinalizeStructs enriches an InspectedPackage with struct constructors and
// reclassifies skipped functions that use known struct types as params/returns.
// ns is the Rugo namespace, pkgAlias is the Go package alias for generated code.
// Must be called before gobridge.Register().
func FinalizeStructs(result *InspectedPackage, ns, pkgAlias string) {
	pkg := result.Package
	if len(pkg.Structs) == 0 {
		return
	}

	// Build a lookup from Go struct name to wrapper type name.
	structWrappers := make(map[string]string) // GoName → wrapper type name
	for _, si := range pkg.Structs {
		structWrappers[si.GoName] = StructWrapperTypeName(ns, si.GoName)
	}

	// Discover methods on struct types before generating wrappers.
	for i := range pkg.Structs {
		si := &pkg.Structs[i]
		named, ok := result.NamedTypes[si.GoName]
		if !ok {
			continue
		}
		si.Methods = discoverMethods(named, structWrappers, result.KnownStructs)
	}

	// Generate wrappers and register constructors (after methods are populated).
	for i, si := range pkg.Structs {
		_ = i
		wrapType := structWrappers[si.GoName]
		helper := GenerateStructWrapper(ns, pkgAlias, si)

		// Register zero-value constructor: mymod.config() → &rugo_struct_mymod_Config{v: &pkg.Config{}}
		constructorName := si.RugoName
		// Avoid collision with existing functions.
		if _, exists := pkg.Funcs[constructorName]; exists {
			constructorName = "new_" + constructorName
		}
		wt := wrapType
		pa := pkgAlias
		gn := si.GoName
		pkg.Funcs[constructorName] = GoFuncSig{
			GoName:  si.GoName,
			Returns: []GoType{GoString}, // placeholder — actual return is opaque handle
			Doc:     fmt.Sprintf("Creates a new zero-value %s struct.", si.GoName),
			Codegen: func(pkgBase string, args []string, rugoName string) string {
				return fmt.Sprintf("interface{}(&%s{v: &%s.%s{}})", wt, pa, gn)
			},
			RuntimeHelpers: []RuntimeHelper{helper},
		}
	}

	// Reclassify skipped functions that are blocked only by known struct pointer types.
	var stillSkipped []ClassifiedFunc
	for _, f := range result.Skipped {
		if f.Tier != TierBlocked {
			stillSkipped = append(stillSkipped, f)
			continue
		}
		sig := reclassifyWithStructs(f, structWrappers, pkgAlias, result.KnownStructs)
		if sig != nil {
			// Attach struct wrapper RuntimeHelpers so they're emitted.
			for _, si := range pkg.Structs {
				wt := structWrappers[si.GoName]
				if needsWrapper(sig, wt) {
					sig.RuntimeHelpers = append(sig.RuntimeHelpers, GenerateStructWrapper(ns, pkgAlias, si))
				}
			}
			pkg.Funcs[f.RugoName] = *sig
		} else {
			stillSkipped = append(stillSkipped, f)
		}
	}
	result.Skipped = stillSkipped
}

// reclassifyWithStructs attempts to build a GoFuncSig for a blocked function
// by resolving struct pointer params/returns to wrapper types.
// Returns nil if the function can't be reclassified.
func reclassifyWithStructs(f ClassifiedFunc, structWrappers map[string]string, pkgAlias string, knownStructs map[string]bool) *GoFuncSig {
	if f.Sig == nil {
		return nil
	}

	sig := &GoFuncSig{
		GoName:   f.GoName,
		Variadic: f.Sig.Variadic(),
	}

	// Classify params.
	params := f.Sig.Params()
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, true)
		if tier == TierBlocked {
			// Check if it's a pointer to a known struct.
			structName := extractStructName(t, knownStructs)
			if structName == "" {
				return nil
			}
			wrapType, ok := structWrappers[structName]
			if !ok {
				return nil
			}
			// Param is an opaque struct handle — use GoString as placeholder type.
			sig.Params = append(sig.Params, GoString)
			if sig.StructCasts == nil {
				sig.StructCasts = make(map[int]string)
			}
			sig.StructCasts[i] = wrapType
		} else if tier == TierFunc {
			return nil // Can't handle func params in reclassification
		} else {
			sig.Params = append(sig.Params, gt)
		}
	}

	// Classify returns.
	results := f.Sig.Results()
	for i := 0; i < results.Len(); i++ {
		t := results.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, false)
		if tier == TierBlocked {
			structName := extractStructName(t, knownStructs)
			if structName == "" {
				return nil
			}
			wrapType, ok := structWrappers[structName]
			if !ok {
				return nil
			}
			// Return is wrapped into an opaque struct handle.
			sig.Returns = append(sig.Returns, GoString) // placeholder
			if sig.StructReturnWraps == nil {
				sig.StructReturnWraps = make(map[int]string)
			}
			sig.StructReturnWraps[i] = wrapType
		} else {
			sig.Returns = append(sig.Returns, gt)
		}
	}

	return sig
}

// extractStructName checks if a type is a pointer to a known struct and returns
// the struct name, or empty string if not.
func extractStructName(t types.Type, knownStructs map[string]bool) string {
	// Handle *Struct (pointer to struct).
	if ptr, ok := t.(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			name := named.Obj().Name()
			if knownStructs[name] {
				return name
			}
		}
	}
	// Handle Struct directly (value type).
	if named, ok := t.(*types.Named); ok {
		name := named.Obj().Name()
		if knownStructs[name] {
			return name
		}
	}
	return ""
}

// needsWrapper checks if a GoFuncSig references a given wrapper type.
func needsWrapper(sig *GoFuncSig, wrapType string) bool {
	for _, w := range sig.StructCasts {
		if w == wrapType {
			return true
		}
	}
	for _, w := range sig.StructReturnWraps {
		if w == wrapType {
			return true
		}
	}
	return false
}

// discoverMethods finds bridgeable methods on a named struct type.
// Uses pointer method set to cover both value and pointer receivers.
func discoverMethods(named *types.Named, structWrappers map[string]string, knownStructs map[string]bool) []GoStructMethodInfo {
	mset := types.NewMethodSet(types.NewPointer(named))
	var methods []GoStructMethodInfo
	for i := 0; i < mset.Len(); i++ {
		sel := mset.At(i)
		fn, ok := sel.Obj().(*types.Func)
		if !ok || !fn.Exported() {
			continue
		}
		sig, ok := fn.Type().(*types.Signature)
		if !ok {
			continue
		}

		mi := classifyMethod(fn.Name(), sig, structWrappers, knownStructs)
		if mi != nil {
			methods = append(methods, *mi)
		}
	}
	return methods
}

// classifyMethod classifies a single method's params and returns (excluding receiver).
// Returns nil if any param/return is unbridgeable.
func classifyMethod(goName string, sig *types.Signature, structWrappers map[string]string, knownStructs map[string]bool) *GoStructMethodInfo {
	mi := &GoStructMethodInfo{
		GoName:   goName,
		RugoName: ToSnakeCase(goName),
		Variadic: sig.Variadic(),
	}

	// Classify params (sig.Params() excludes the receiver for methods).
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		t := params.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, true)
		if tier == TierBlocked {
			structName := extractStructName(t, knownStructs)
			if structName == "" {
				return nil
			}
			wrapType, ok := structWrappers[structName]
			if !ok {
				return nil
			}
			mi.Params = append(mi.Params, GoString) // placeholder
			if mi.StructCasts == nil {
				mi.StructCasts = make(map[int]string)
			}
			mi.StructCasts[i] = wrapType
		} else if tier == TierFunc {
			return nil
		} else {
			mi.Params = append(mi.Params, gt)
		}
	}

	// Classify returns.
	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		t := results.At(i).Type()
		gt, tier, _ := ClassifyGoType(t, false)
		if tier == TierBlocked {
			structName := extractStructName(t, knownStructs)
			if structName == "" {
				return nil
			}
			wrapType, ok := structWrappers[structName]
			if !ok {
				return nil
			}
			mi.Returns = append(mi.Returns, GoString) // placeholder
			if mi.StructReturnWraps == nil {
				mi.StructReturnWraps = make(map[int]string)
			}
			mi.StructReturnWraps[i] = wrapType
			// Track if the return is a value type (not pointer) — needs &addr.
			if _, isPtr := t.(*types.Pointer); !isPtr {
				if mi.StructReturnValue == nil {
					mi.StructReturnValue = make(map[int]bool)
				}
				mi.StructReturnValue[i] = true
			}
		} else {
			mi.Returns = append(mi.Returns, gt)
		}
	}

	return mi
}
