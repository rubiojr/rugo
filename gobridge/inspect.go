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
	scope := typePkg.Scope()
	for _, name := range scope.Names() {
		obj := scope.Lookup(name)

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

	if len(funcs) == 0 {
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
	}

	return &InspectedPackage{
		Package:      pkg,
		GoModulePath: modulePath,
		Skipped:      skipped,
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
