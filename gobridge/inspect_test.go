package gobridge

import (
	"path/filepath"
	"runtime"
	"testing"
)

func fixtureDir(name string) string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "testdata", name)
}

func TestInspectSourcePackage(t *testing.T) {
	dir := fixtureDir("fixture_gomod")
	result, err := InspectSourcePackage(dir)
	if err != nil {
		t.Fatalf("InspectSourcePackage: %v", err)
	}

	if result.GoModulePath != "example.com/testmod" {
		t.Errorf("GoModulePath = %q, want %q", result.GoModulePath, "example.com/testmod")
	}

	pkg := result.Package
	if pkg.Path != "example.com/testmod" {
		t.Errorf("Package.Path = %q, want %q", pkg.Path, "example.com/testmod")
	}

	// Should have 3 bridgeable functions (unexported `helper` must not appear)
	wantFuncs := map[string]string{
		"greet":   "Greet",
		"add":     "Add",
		"is_even": "IsEven",
	}
	if len(pkg.Funcs) != len(wantFuncs) {
		t.Errorf("Funcs count = %d, want %d", len(pkg.Funcs), len(wantFuncs))
	}
	for rugoName, goName := range wantFuncs {
		sig, ok := pkg.Funcs[rugoName]
		if !ok {
			t.Errorf("missing function %q", rugoName)
			continue
		}
		if sig.GoName != goName {
			t.Errorf("Funcs[%q].GoName = %q, want %q", rugoName, sig.GoName, goName)
		}
	}

	// Verify param/return types
	greet := pkg.Funcs["greet"]
	if len(greet.Params) != 1 || greet.Params[0] != GoString {
		t.Errorf("greet params = %v, want [GoString]", greet.Params)
	}
	if len(greet.Returns) != 1 || greet.Returns[0] != GoString {
		t.Errorf("greet returns = %v, want [GoString]", greet.Returns)
	}

	add := pkg.Funcs["add"]
	if len(add.Params) != 2 || add.Params[0] != GoInt || add.Params[1] != GoInt {
		t.Errorf("add params = %v, want [GoInt, GoInt]", add.Params)
	}

	// Should have 2 skipped functions (WithPointer, MakeChan)
	if len(result.Skipped) != 2 {
		t.Errorf("skipped = %d, want 2", len(result.Skipped))
		for _, s := range result.Skipped {
			t.Logf("  skipped: %s (%s)", s.GoName, s.Reason)
		}
	}
}

func TestInspectSourcePackage_NoGoMod(t *testing.T) {
	_, err := InspectSourcePackage(t.TempDir())
	if err == nil {
		t.Error("expected error for directory without go.mod")
	}
}

func TestIsGoModuleDir(t *testing.T) {
	if !IsGoModuleDir(fixtureDir("fixture_gomod")) {
		t.Error("expected fixture_gomod to be a Go module dir")
	}
	if IsGoModuleDir(t.TempDir()) {
		t.Error("expected temp dir to not be a Go module dir")
	}
}

func TestReadGoModulePath(t *testing.T) {
	path := filepath.Join(fixtureDir("fixture_gomod"), "go.mod")
	mod, err := ReadGoModulePath(path)
	if err != nil {
		t.Fatalf("ReadGoModulePath: %v", err)
	}
	if mod != "example.com/testmod" {
		t.Errorf("module path = %q, want %q", mod, "example.com/testmod")
	}
}
