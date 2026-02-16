package gobridge

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io"
	"os"
	"os/exec"
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
	// ExternalTypes maps qualified keys (pkgPath.TypeName) to external type info.
	// Populated by FinalizeStructs when blocked functions reference types from
	// external packages.
	ExternalTypes map[string]ExternalTypeInfo
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
	// Use module-aware importer when available (resolves external dependencies
	// via `go list -export`), falling back to default for stdlib-only packages.
	conf := types.Config{
		Importer: moduleAwareImporter(fset, dir),
		Error:    func(error) {},
	}
	typePkg, _ := conf.Check(pkgName, fset, files, nil)
	if typePkg == nil {
		return nil, fmt.Errorf("type checking failed for %s", modulePath)
	}

	cr := classifyScope(typePkg.Scope(), true)

	if len(cr.Funcs) == 0 && len(cr.Structs) == 0 && len(cr.Skipped) == 0 {
		return nil, fmt.Errorf("no bridgeable functions found in %s", modulePath)
	}

	pkg := &Package{
		Path:     modulePath,
		Funcs:    cr.Funcs,
		Doc:      fmt.Sprintf("Functions from Go module %s.", modulePath),
		External: true,
		Structs:  cr.Structs,
	}

	return &InspectedPackage{
		Package:      pkg,
		GoModulePath: modulePath,
		Skipped:      cr.Skipped,
		KnownStructs: cr.KnownStructs,
		NamedTypes:   cr.NamedTypes,
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

// moduleAwareImporter returns a types.Importer that resolves imports using
// `go list -export -json`, which is module-aware and handles dependencies
// in the Go module cache. Falls back to importer.Default() for stdlib.
// dir is the directory of the Go module being inspected (used as cwd for go list).
func moduleAwareImporter(fset *token.FileSet, dir string) types.Importer {
	// Check if `go list` works in this directory (has go.mod with dependencies).
	// If not, fall back to the default importer (works for stdlib-only packages).
	goModDir, found := FindGoModDir(dir)
	if !found {
		return importer.Default()
	}

	return importerFunc(func(path string) (*types.Package, error) {
		// Try module-aware import via `go list -export`.
		cmd := exec.Command("go", "list", "-export", "-json", path)
		cmd.Dir = goModDir
		out, err := cmd.Output()
		if err != nil {
			// Fall back to default importer for stdlib packages.
			return importer.Default().Import(path)
		}

		var info struct{ Export string }
		if err := json.Unmarshal(out, &info); err != nil || info.Export == "" {
			return importer.Default().Import(path)
		}

		// Use ForCompiler with a lookup that reads the export file.
		gcImp := importer.ForCompiler(fset, "gc", func(p string) (io.ReadCloser, error) {
			if p == path {
				return os.Open(info.Export)
			}
			// For transitive imports, delegate to go list as well.
			cmd := exec.Command("go", "list", "-export", "-json", p)
			cmd.Dir = goModDir
			out, err := cmd.Output()
			if err != nil {
				return nil, fmt.Errorf("go list %s: %w", p, err)
			}
			var info2 struct{ Export string }
			if err := json.Unmarshal(out, &info2); err != nil || info2.Export == "" {
				return nil, fmt.Errorf("no export data for %s", p)
			}
			return os.Open(info2.Export)
		})
		return gcImp.Import(path)
	})
}

// importerFunc adapts a function to the types.Importer interface.
type importerFunc func(path string) (*types.Package, error)

func (f importerFunc) Import(path string) (*types.Package, error) { return f(path) }

// classifiedScope holds the results of classifying all exported symbols in a Go package scope.
type classifiedScope struct {
	Funcs        map[string]GoFuncSig
	Skipped      []ClassifiedFunc
	Structs      []GoStructInfo
	KnownStructs map[string]bool
	NamedTypes   map[string]*types.Named
}

// classifyScope enumerates exported symbols from a Go package scope and classifies them.
// When discoverStructs is true, exported struct types are discovered for wrapper generation
// (used by InspectSourcePackage for require'd Go modules).
// Package-level var methods (e.g., base64.StdEncoding.EncodeToString) are always discovered.
func classifyScope(scope *types.Scope, discoverStructs bool) classifiedScope {
	var allFuncs []ClassifiedFunc
	var structInfos []GoStructInfo
	knownStructs := make(map[string]bool)
	namedTypes := make(map[string]*types.Named)
	varConsts := make(map[string]GoFuncSig)

	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

		// Discover exported struct types (for require'd Go modules).
		if discoverStructs {
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
		}

		// Package-level functions.
		if fn, ok := obj.(*types.Func); ok {
			if !fn.Exported() {
				continue
			}
			sig := fn.Type().(*types.Signature)
			if sig.Recv() != nil {
				continue
			}
			bf := ClassifyFunc(name, ToSnakeCase(name), sig)
			allFuncs = append(allFuncs, bf)
			continue
		}

		// Package-level vars — enumerate methods and bridge value if type is bridgeable.
		if v, ok := obj.(*types.Var); ok && v.Exported() {
			allFuncs = append(allFuncs, classifyVarMethods(name, v)...)
			if sig := classifyVarValue(name, v); sig != nil {
				varConsts[ToSnakeCase(name)] = *sig
			}
		}

		// Package-level consts — bridge value if type is bridgeable.
		if c, ok := obj.(*types.Const); ok && c.Exported() {
			if sig := classifyConstValue(name, c); sig != nil {
				varConsts[ToSnakeCase(name)] = *sig
			}
		}
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
			if len(f.TypeCasts) > 0 {
				sig.TypeCasts = f.TypeCasts
			}
			funcs[f.RugoName] = sig
		} else {
			skipped = append(skipped, f)
		}
	}

	// Merge var/const accessors (don't overwrite function entries).
	for name, sig := range varConsts {
		if _, exists := funcs[name]; !exists {
			funcs[name] = sig
		}
	}

	// Auto-wrap output-buffer functions: detect funcs where the first param
	// is a write-destination []byte and a companion sizing function exists.
	// E.g., hex.Encode(dst, src []byte) int → auto-allocates dst via EncodedLen.
	autoWrapDstBufferFuncs(funcs)

	return classifiedScope{
		Funcs:        funcs,
		Skipped:      skipped,
		Structs:      structInfos,
		KnownStructs: knownStructs,
		NamedTypes:   namedTypes,
	}
}

// autoWrapDstBufferFuncs detects Go functions with output-buffer params and
// replaces them with auto-wrapping Codegen callbacks.
//
// Pattern: func Name(dst, src []byte) int — writes into dst, returns byte count.
// The wrapper removes dst from the Rugo signature, auto-allocates it using a
// companion NamedLen() function, and returns the filled buffer as a string.
//
// Examples:
//
//	hex.Encode(dst, src []byte) int        → hex.encode(src) returns string
//	hex.Decode(dst, src []byte) (int, error) → hex.decode(src) returns string
func autoWrapDstBufferFuncs(funcs map[string]GoFuncSig) {
	for rugoName, sig := range funcs {
		if !isDstBufferFunc(sig) {
			continue
		}
		lenFunc := findCompanionLenFunc(sig.GoName, funcs)
		if lenFunc == "" {
			continue
		}
		hasError := len(sig.Returns) == 2 && sig.Returns[1] == GoError

		// Capture for closure
		goName := sig.GoName
		lenFuncName := lenFunc

		wrapped := sig
		wrapped.Params = sig.Params[1:] // remove dst param
		wrapped.Returns = []GoType{GoByteSlice}
		wrapped.Codegen = func(pkgBase string, args []string, rugoName string) string {
			srcExpr := TypeConvToGo(args[0], GoByteSlice)
			call := fmt.Sprintf("%s.%s", pkgBase, goName)
			lenCall := fmt.Sprintf("%s.%s(len(_src))", pkgBase, lenFuncName)
			if hasError {
				return fmt.Sprintf("func() interface{} { _src := %s; _dst := make([]byte, %s); _n, _err := %s(_dst, _src); if _err != nil { %s }; return interface{}(string(_dst[:_n])) }()",
					srcExpr, lenCall, call, PanicOnErr(rugoName))
			}
			return fmt.Sprintf("func() interface{} { _src := %s; _dst := make([]byte, %s); %s(_dst, _src); return interface{}(string(_dst)) }()",
				srcExpr, lenCall, call)
		}
		funcs[rugoName] = wrapped
	}
}

// isDstBufferFunc returns true if the function matches the output-buffer pattern:
// first param is GoByteSlice, second param is GoByteSlice, and the function
// returns int or (int, error) — not []byte.
func isDstBufferFunc(sig GoFuncSig) bool {
	if len(sig.Params) < 2 || sig.Params[0] != GoByteSlice || sig.Params[1] != GoByteSlice {
		return false
	}
	if len(sig.Returns) == 0 {
		return false
	}
	// Must return int (byte count), not []byte
	if sig.Returns[0] != GoInt {
		return false
	}
	if len(sig.Returns) == 1 {
		return true
	}
	// (int, error) is OK
	return len(sig.Returns) == 2 && sig.Returns[1] == GoError
}

// findCompanionLenFunc finds a companion sizing function for an output-buffer
// function. For example, "Encode" → "EncodedLen", "Decode" → "DecodedLen".
func findCompanionLenFunc(goName string, funcs map[string]GoFuncSig) string {
	// Pattern: Name → NamedLen (e.g., Encode → EncodedLen)
	candidate := goName + "dLen"
	for _, sig := range funcs {
		if sig.GoName == candidate {
			return candidate
		}
	}
	// Pattern: Name → NameLen (e.g., Encode → EncodeLen — less common)
	candidate = goName + "Len"
	for _, sig := range funcs {
		if sig.GoName == candidate {
			return candidate
		}
	}
	return ""
}

// classifyVarMethods enumerates bridgeable methods on an exported package-level var.
// For example, base64.StdEncoding has methods like EncodeToString, DecodeString, etc.
// Returns classified functions with dot-chain GoNames (e.g., "StdEncoding.EncodeToString").
func classifyVarMethods(varName string, v *types.Var) []ClassifiedFunc {
	varType := v.Type()
	if ptr, ok := varType.(*types.Pointer); ok {
		varType = ptr.Elem()
	}
	named, ok := varType.(*types.Named)
	if !ok {
		return nil
	}

	seen := make(map[string]bool)
	var funcs []ClassifiedFunc

	// Value receiver methods.
	for i := 0; i < named.NumMethods(); i++ {
		m := named.Method(i)
		if !m.Exported() {
			continue
		}
		goName := varName + "." + m.Name()
		seen[goName] = true
		rugoName := ToSnakeCase(varName) + "_" + ToSnakeCase(m.Name())
		bf := ClassifyFunc(goName, rugoName, m.Type().(*types.Signature))
		funcs = append(funcs, bf)
	}

	// Pointer receiver methods (deduped).
	ptrMethods := types.NewMethodSet(types.NewPointer(named))
	for i := 0; i < ptrMethods.Len(); i++ {
		m := ptrMethods.At(i).Obj().(*types.Func)
		if !m.Exported() {
			continue
		}
		goName := varName + "." + m.Name()
		if seen[goName] {
			continue
		}
		rugoName := ToSnakeCase(varName) + "_" + ToSnakeCase(m.Name())
		bf := ClassifyFunc(goName, rugoName, m.Type().(*types.Signature))
		funcs = append(funcs, bf)
	}

	return funcs
}

// classifyVarValue bridges an exported var as a zero-arg accessor if its type
// is bridgeable. For example, os.Args ([]string) is accessed as os.args.
func classifyVarValue(name string, v *types.Var) *GoFuncSig {
	gt, tier, _ := ClassifyGoType(v.Type(), false)
	if tier == TierBlocked || tier == TierFunc {
		return nil
	}
	goName := name
	return &GoFuncSig{
		GoName:  goName,
		Returns: []GoType{gt},
		Doc:     fmt.Sprintf("Variable %s.", name),
		Codegen: func(pkgBase string, _ []string, _ string) string {
			return TypeWrapReturn(pkgBase+"."+goName, gt)
		},
	}
}

// classifyConstValue bridges an exported const as a zero-arg accessor if its
// type is bridgeable. For example, math.Pi is accessed as math.pi.
func classifyConstValue(name string, c *types.Const) *GoFuncSig {
	gt, tier, _ := ClassifyGoType(c.Type(), false)
	if tier == TierBlocked || tier == TierFunc {
		return nil
	}
	goName := name
	return &GoFuncSig{
		GoName:  goName,
		Returns: []GoType{gt},
		Doc:     fmt.Sprintf("Constant %s.", name),
		Codegen: func(pkgBase string, _ []string, _ string) string {
			return TypeWrapReturn(pkgBase+"."+goName, gt)
		},
	}
}

// InspectCompiledPackage introspects a compiled Go package (stdlib or installed)
// using importer.Default() and returns a bridge Package ready for registration.
// This is the compile-time equivalent of InspectSourcePackage — used by `import`
// to dynamically bridge any Go stdlib package without pre-generated code.
func InspectCompiledPackage(pkgPath string) (*Package, error) {
	pkg, err := importer.Default().Import(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("importing %s: %w", pkgPath, err)
	}

	cr := classifyScope(pkg.Scope(), false)

	if len(cr.Funcs) == 0 {
		return nil, fmt.Errorf("no bridgeable functions in %s", pkgPath)
	}

	return &Package{
		Path:  pkgPath,
		Funcs: cr.Funcs,
		Doc:   fmt.Sprintf("Functions from Go's %s package.", pkgPath),
	}, nil
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
// It also discovers external named types from dependencies and generates
// opaque wrappers for them, enabling functions that use external types to be bridged.
// ns is the Rugo namespace, pkgAlias is the Go package alias for generated code.
// Must be called before gobridge.Register().
func FinalizeStructs(result *InspectedPackage, ns, pkgAlias string) {
	pkg := result.Package

	// Build a lookup from Go struct name to wrapper type name.
	structWrappers := make(map[string]string) // GoName or qualified key → wrapper type name

	if len(pkg.Structs) > 0 {
		for _, si := range pkg.Structs {
			structWrappers[si.GoName] = StructWrapperTypeName(ns, si.GoName)
		}

		// Discover methods and embedded fields on struct types before generating wrappers.
		for i := range pkg.Structs {
			si := &pkg.Structs[i]
			named, ok := result.NamedTypes[si.GoName]
			if !ok {
				continue
			}
			si.Methods = discoverMethods(named, structWrappers, result.KnownStructs)
			// Discover embedded pointer-to-struct fields for upcast support.
			embedded := discoverEmbeddedFields(named, structWrappers, result.KnownStructs)
			si.Fields = append(si.Fields, embedded...)
		}

		// Generate wrappers and register constructors (after methods are populated).
		for _, si := range pkg.Structs {
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
	}

	// Discover external named types from blocked function signatures.
	externalTypes := discoverExternalTypes(result, pkg.Path)

	// Recursively discover types embedded in already-known external types.
	// This handles class hierarchies like QPushButton → *QAbstractButton → *QWidget.
	for changed := true; changed; {
		changed = false
		for _, ext := range externalTypes {
			if ext.Named == nil {
				continue
			}
			st, ok := ext.Named.Underlying().(*types.Struct)
			if !ok {
				continue
			}
			for i := 0; i < st.NumFields(); i++ {
				f := st.Field(i)
				if !f.Exported() || !f.Embedded() {
					continue
				}
				collectExternalFromType(f.Type(), pkg.Path, result.KnownStructs, externalTypes)
			}
		}
		// Register any new types found in this pass.
		for key, ext := range externalTypes {
			if !result.KnownStructs[key] {
				result.KnownStructs[key] = true
				changed = true
				_ = ext // will be registered below
			}
		}
	}

	result.ExternalTypes = externalTypes
	for key, ext := range externalTypes {
		wrapType := ExternalOpaqueWrapperTypeName(ns, ext.PkgName, ext.GoName)
		structWrappers[key] = wrapType
		result.KnownStructs[key] = true
		// Track the import path so codegen emits it.
		if !containsString(pkg.ExtraImports, ext.PkgPath) {
			pkg.ExtraImports = append(pkg.ExtraImports, ext.PkgPath)
		}
	}

	// Discover methods and embedded fields on external types (after all types are in structWrappers).
	for key, ext := range externalTypes {
		if ext.Named == nil {
			continue
		}
		ext.Methods = discoverMethods(ext.Named, structWrappers, result.KnownStructs)
		ext.EmbeddedFields = discoverEmbeddedFields(ext.Named, structWrappers, result.KnownStructs)
		externalTypes[key] = ext
	}

	// Pre-collect all external type wrappers as RuntimeHelpers.
	// These are emitted once (deduped by key) and cover the full type hierarchy
	// including intermediate types only referenced via embedded fields.
	var allExternalHelpers []RuntimeHelper
	for _, ext := range externalTypes {
		allExternalHelpers = append(allExternalHelpers, GenerateExternalOpaqueWrapper(ns, ext))
		allExternalHelpers = append(allExternalHelpers, methodRuntimeHelpers(ext.Methods)...)
	}
	// Generate upcast helpers for all wrapper types (enables auto-upcasting).
	for _, wrapType := range structWrappers {
		allExternalHelpers = append(allExternalHelpers, GenerateUpcastHelper(wrapType))
	}

	// Reclassify skipped functions that are blocked only by known struct/external pointer types.
	var stillSkipped []ClassifiedFunc
	for _, f := range result.Skipped {
		if f.Tier != TierBlocked {
			stillSkipped = append(stillSkipped, f)
			continue
		}
		sig := reclassifyWithStructs(f, structWrappers, pkgAlias, result.KnownStructs)
		if sig != nil {
			// Attach in-package struct wrapper RuntimeHelpers.
			for _, si := range pkg.Structs {
				wt := structWrappers[si.GoName]
				if needsWrapper(sig, wt) {
					sig.RuntimeHelpers = append(sig.RuntimeHelpers, GenerateStructWrapper(ns, pkgAlias, si))
				}
			}
			// Attach all external type wrappers (deduped by key in codegen).
			if len(allExternalHelpers) > 0 {
				sig.RuntimeHelpers = append(sig.RuntimeHelpers, allExternalHelpers...)
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
// Also handles GoFunc params alongside struct params (func+struct combo).
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
			// Check for string view types before the struct wrapper path.
			if ctor := stringViewConstructor(t); ctor != "" {
				sig.Params = append(sig.Params, GoString)
				if sig.TypeCasts == nil {
					sig.TypeCasts = make(map[int]string)
				}
				sig.TypeCasts[i] = ctor
				continue
			}
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
			// Classify the function parameter for GoFunc adapter.
			funcSig, ok := params.At(i).Type().Underlying().(*types.Signature)
			if !ok {
				return nil
			}
			ft := ClassifyFuncType(funcSig, structWrappers, knownStructs)
			if ft == nil {
				return nil
			}
			sig.Params = append(sig.Params, GoFunc)
			if sig.FuncTypes == nil {
				sig.FuncTypes = make(map[int]*GoFuncType)
			}
			sig.FuncTypes[i] = ft
		} else {
			sig.Params = append(sig.Params, gt)
			// Detect named types that need explicit casts (e.g., qt6.StandardKey).
			if cast := namedTypeCast(t); cast != "" {
				if sig.TypeCasts == nil {
					sig.TypeCasts = make(map[int]string)
				}
				sig.TypeCasts[i] = cast
			}
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
// the struct name (or qualified key for external types), or empty string if not.
func extractStructName(t types.Type, knownStructs map[string]bool) string {
	// Handle *Struct (pointer to struct).
	if ptr, ok := t.(*types.Pointer); ok {
		if named, ok := ptr.Elem().(*types.Named); ok {
			name := named.Obj().Name()
			if knownStructs[name] {
				return name
			}
			// Check qualified key for external types.
			if pkg := named.Obj().Pkg(); pkg != nil {
				qualKey := ExternalTypeKey(pkg.Path(), name)
				if knownStructs[qualKey] {
					return qualKey
				}
			}
		}
	}
	// Handle Struct directly (value type).
	if named, ok := t.(*types.Named); ok {
		name := named.Obj().Name()
		if knownStructs[name] {
			return name
		}
		if pkg := named.Obj().Pkg(); pkg != nil {
			qualKey := ExternalTypeKey(pkg.Path(), name)
			if knownStructs[qualKey] {
				return qualKey
			}
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

// stringViewConstructor checks if a type is a named struct whose name contains
// "StringView" (e.g. Qt6's QAnyStringView). These are value-type wrappers
// around strings that should be auto-converted from Rugo strings using
// their NewXxx(string) constructor.
//
// Returns the qualified constructor expression (e.g. "qt6.NewQAnyStringView3")
// or "" if the type is not a string view.
func stringViewConstructor(t types.Type) string {
	named, ok := t.(*types.Named)
	if !ok {
		return ""
	}
	if _, isStruct := named.Underlying().(*types.Struct); !isStruct {
		return ""
	}
	typeName := named.Obj().Name()
	if !strings.Contains(typeName, "StringView") {
		return ""
	}

	// Find a constructor that takes a single string param.
	// Convention: NewTypeName(string) or NewTypeName2(string), etc.
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return ""
	}
	scope := pkg.Scope()
	prefix := "New" + typeName
	for _, name := range scope.Names() {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		obj := scope.Lookup(name)
		fn, ok := obj.(*types.Func)
		if !ok {
			continue
		}
		sig := fn.Type().(*types.Signature)
		if sig.Params().Len() != 1 || sig.Results().Len() != 1 {
			continue
		}
		paramType := sig.Params().At(0).Type()
		if b, ok := paramType.Underlying().(*types.Basic); ok && b.Kind() == types.String {
			ctor := pkg.Name() + "." + name
			// If constructor returns a pointer, prefix with * so codegen dereferences.
			retType := sig.Results().At(0).Type()
			if _, isPtr := retType.(*types.Pointer); isPtr {
				ctor = "*" + ctor
			}
			return ctor
		}
	}
	return ""
}

// discoverEmbeddedFields finds embedded pointer-to-struct fields on a named type
// that point to known struct types. Returns field info with WrapType set.
func discoverEmbeddedFields(named *types.Named, structWrappers map[string]string, knownStructs map[string]bool) []GoStructFieldInfo {
	st, ok := named.Underlying().(*types.Struct)
	if !ok {
		return nil
	}
	var fields []GoStructFieldInfo
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() || !f.Embedded() {
			continue
		}
		// Check if the embedded field is a pointer to a known struct.
		structName := extractStructName(f.Type(), knownStructs)
		if structName == "" {
			continue
		}
		wrapType, ok := structWrappers[structName]
		if !ok {
			continue
		}
		fields = append(fields, GoStructFieldInfo{
			GoName:   f.Name(),
			RugoName: ToSnakeCase(f.Name()),
			WrapType: wrapType,
		})
	}
	return fields
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
			// Check for string view types (e.g., QAnyStringView) before
			// falling back to the struct wrapper path. These are value-type
			// structs that wrap a string — treat as GoString with a
			// constructor-based TypeCast.
			if ctor := stringViewConstructor(t); ctor != "" {
				mi.Params = append(mi.Params, GoString)
				if mi.TypeCasts == nil {
					mi.TypeCasts = make(map[int]string)
				}
				mi.TypeCasts[i] = ctor
				continue
			}
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
			// Track if the param is a value type (not pointer) — needs dereference.
			if _, isPtr := t.(*types.Pointer); !isPtr {
				if mi.StructParamValue == nil {
					mi.StructParamValue = make(map[int]bool)
				}
				mi.StructParamValue[i] = true
			}
		} else if tier == TierFunc {
			// Classify the function parameter for GoFunc adapter.
			funcSig, ok := params.At(i).Type().Underlying().(*types.Signature)
			if !ok {
				return nil
			}
			ft := ClassifyFuncType(funcSig, structWrappers, knownStructs)
			if ft == nil {
				return nil
			}
			mi.Params = append(mi.Params, GoFunc)
			if mi.FuncTypes == nil {
				mi.FuncTypes = make(map[int]*GoFuncType)
			}
			mi.FuncTypes[i] = ft
		} else {
			mi.Params = append(mi.Params, gt)
			// Detect named types that need explicit casts (e.g., qt6.GestureType).
			if cast := namedTypeCast(t); cast != "" {
				if mi.TypeCasts == nil {
					mi.TypeCasts = make(map[int]string)
				}
				mi.TypeCasts[i] = cast
			}
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

// discoverExternalTypes scans blocked function signatures for pointer-to-named
// types from external packages (types not defined in the inspected module).
// Returns a map keyed by qualified name (pkgPath.TypeName) to ExternalTypeInfo.
func discoverExternalTypes(result *InspectedPackage, modulePath string) map[string]ExternalTypeInfo {
	externals := make(map[string]ExternalTypeInfo)

	for _, f := range result.Skipped {
		if f.Sig == nil || f.Tier != TierBlocked {
			continue
		}
		collectExternalFromSig(f.Sig, modulePath, result.KnownStructs, externals)
	}

	return externals
}

// collectExternalFromSig examines a function signature for external named types.
func collectExternalFromSig(sig *types.Signature, modulePath string, knownStructs map[string]bool, externals map[string]ExternalTypeInfo) {
	params := sig.Params()
	for i := 0; i < params.Len(); i++ {
		collectExternalFromType(params.At(i).Type(), modulePath, knownStructs, externals)
	}
	results := sig.Results()
	for i := 0; i < results.Len(); i++ {
		collectExternalFromType(results.At(i).Type(), modulePath, knownStructs, externals)
	}
}

// collectExternalFromType checks if a type is a pointer (or value) of a named
// type from an external package and adds it to the externals map.
func collectExternalFromType(t types.Type, modulePath string, knownStructs map[string]bool, externals map[string]ExternalTypeInfo) {
	var named *types.Named

	if ptr, ok := t.(*types.Pointer); ok {
		named, _ = ptr.Elem().(*types.Named)
	} else {
		named, _ = t.(*types.Named)
	}
	if named == nil {
		return
	}

	pkg := named.Obj().Pkg()
	if pkg == nil {
		return // built-in type (error, etc.)
	}

	typeName := named.Obj().Name()

	// Skip types already known as in-package structs.
	if knownStructs[typeName] {
		return
	}

	// Skip types from the module itself — those are handled as in-package structs.
	// Check both the full module path and the short package name (the type checker
	// may use either depending on whether dependencies were resolved).
	if pkg.Path() == modulePath || pkg.Name() == DefaultNS(modulePath) {
		return
	}

	key := ExternalTypeKey(pkg.Path(), typeName)
	if _, exists := externals[key]; exists {
		return
	}

	externals[key] = ExternalTypeInfo{
		PkgPath: pkg.Path(),
		PkgName: pkg.Name(),
		GoName:  typeName,
		Named:   named,
	}
}

func containsString(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

// namedTypeCast returns the qualified Go type expression for a named type
// that needs an explicit cast (e.g., "qt6.GestureType"). Returns empty
// string if the type is a basic type or doesn't need casting.
func namedTypeCast(t types.Type) string {
	named, ok := t.(*types.Named)
	if !ok {
		return basicTypeCast(t)
	}
	pkg := named.Obj().Pkg()
	if pkg == nil {
		return "" // built-in (error, etc.)
	}
	// Only emit cast for types with a different underlying basic type.
	if _, isBasic := named.Underlying().(*types.Basic); !isBasic {
		return ""
	}
	return pkg.Name() + "." + named.Obj().Name()
}

// basicTypeCast returns a Go type cast for basic types that need narrowing
// (e.g., int8, int16, uint16). Returns empty string if no cast is needed.
func basicTypeCast(t types.Type) string {
	b, ok := t.(*types.Basic)
	if !ok {
		return ""
	}
	switch b.Kind() {
	case types.Int8:
		return "int8"
	case types.Int16:
		return "int16"
	case types.Uint16:
		return "uint16"
	default:
		return ""
	}
}

// methodRuntimeHelpers returns any runtime helpers needed by the given methods
// (e.g., rune helper if any method uses GoRune params/returns).
func methodRuntimeHelpers(methods []GoStructMethodInfo) []RuntimeHelper {
	var helpers []RuntimeHelper
	needsRune := false
	needsStringSlice := false
	for _, m := range methods {
		for _, p := range m.Params {
			if p == GoRune {
				needsRune = true
			}
			if p == GoStringSlice {
				needsStringSlice = true
			}
		}
		for _, r := range m.Returns {
			if r == GoRune {
				needsRune = true
			}
			if r == GoStringSlice {
				needsStringSlice = true
			}
		}
	}
	if needsRune {
		helpers = append(helpers, RuneHelper)
	}
	if needsStringSlice {
		helpers = append(helpers, StringSliceHelper)
	}
	return helpers
}
