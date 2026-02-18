package gobridge

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestStringViewParamClassifiedAsString(t *testing.T) {
	dir := fixtureDir("fixture_stringview")
	result, err := InspectSourcePackage(dir)
	if err != nil {
		t.Fatalf("InspectSourcePackage: %v", err)
	}

	// FinalizeStructs discovers methods (including those with StringView params).
	FinalizeStructs(result, "stringview", "stringview")

	// Find the Widget struct info.
	var widgetInfo *GoStructInfo
	for i := range result.Package.Structs {
		if result.Package.Structs[i].GoName == "Widget" {
			widgetInfo = &result.Package.Structs[i]
			break
		}
	}
	if widgetInfo == nil {
		t.Fatal("Widget struct not found")
	}

	// SetObjectName takes AnyStringView — should be bridged, not blocked.
	var setObjName *GoStructMethodInfo
	for i := range widgetInfo.Methods {
		if widgetInfo.Methods[i].GoName == "SetObjectName" {
			setObjName = &widgetInfo.Methods[i]
			break
		}
	}
	if setObjName == nil {
		t.Fatal("SetObjectName method not found — was it blocked by AnyStringView param?")
	}

	// The param should be classified as GoString.
	if len(setObjName.Params) != 1 {
		t.Fatalf("SetObjectName params: got %d, want 1", len(setObjName.Params))
	}
	if setObjName.Params[0] != GoString {
		t.Errorf("SetObjectName param[0]: got %v, want GoString", setObjName.Params[0])
	}

	// Should have a TypeCast for the constructor conversion (not a StructCast).
	if setObjName.StructCasts != nil {
		t.Errorf("SetObjectName should not have StructCasts, got %v", setObjName.StructCasts)
	}
	if setObjName.TypeCasts == nil || setObjName.TypeCasts[0] == "" {
		t.Error("SetObjectName should have TypeCasts[0] for AnyStringView constructor")
	} else {
		// Fixture constructor returns value type — no * prefix.
		assert.Equal(t, "stringview.NewAnyStringView", setObjName.TypeCasts[0])
	}

	// SetTitle takes plain string — should work as before.
	var setTitle *GoStructMethodInfo
	for i := range widgetInfo.Methods {
		if widgetInfo.Methods[i].GoName == "SetTitle" {
			setTitle = &widgetInfo.Methods[i]
			break
		}
	}
	if setTitle == nil {
		t.Fatal("SetTitle method not found")
	}
	if len(setTitle.Params) != 1 || setTitle.Params[0] != GoString {
		t.Errorf("SetTitle params: got %v, want [GoString]", setTitle.Params)
	}
	if setTitle.TypeCasts != nil {
		t.Errorf("SetTitle should not have TypeCasts, got %v", setTitle.TypeCasts)
	}

	// SetTooltip takes PtrStringView (constructor returns pointer) — should
	// have a * prefix on the TypeCast so codegen dereferences.
	var setTooltip *GoStructMethodInfo
	for i := range widgetInfo.Methods {
		if widgetInfo.Methods[i].GoName == "SetTooltip" {
			setTooltip = &widgetInfo.Methods[i]
			break
		}
	}
	if setTooltip == nil {
		t.Fatal("SetTooltip method not found — was it blocked by PtrStringView param?")
	}
	if len(setTooltip.Params) != 1 || setTooltip.Params[0] != GoString {
		t.Errorf("SetTooltip params: got %v, want [GoString]", setTooltip.Params)
	}
	if setTooltip.TypeCasts == nil {
		t.Fatal("SetTooltip should have TypeCasts[0] for PtrStringView constructor")
	}
	assert.Equal(t, "*stringview.NewPtrStringView", setTooltip.TypeCasts[0])
}

func TestInspectCompiledPackage_TimeStructSupport(t *testing.T) {
	pkg, err := InspectCompiledPackage("time")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	now, ok := pkg.Funcs["now"]
	require.True(t, ok, "time.now should be bridged")
	require.Contains(t, now.StructReturnWraps, 0, "time.now should wrap time.Time return")
	assert.True(t, now.StructReturnValue[0], "time.now returns value type time.Time")
	assert.NotEmpty(t, now.RuntimeHelpers, "time.now should carry wrapper helpers")

	since, ok := pkg.Funcs["since"]
	require.True(t, ok, "time.since should be bridged")
	require.Contains(t, since.StructCasts, 0, "time.since should unwrap time.Time param")
	assert.True(t, since.StructParamValue[0], "time.since takes value type time.Time")
}
