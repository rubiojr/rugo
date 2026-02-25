package gobridge

import (
	"errors"
	"go/token"
	"go/types"
	"os"
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

func TestInspectCompiledPackage_EncodingXMLNewDecoder(t *testing.T) {
	pkg, err := InspectCompiledPackage("encoding/xml")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	sig, ok := pkg.Funcs["new_decoder"]
	require.True(t, ok, "xml.new_decoder should be bridged")
	require.Len(t, sig.Params, 1)
	assert.Equal(t, GoAny, sig.Params[0])
	require.Contains(t, sig.TypeCasts, 0)
	assert.Equal(t, "assert:interface{Read([]byte) (int, error)}", sig.TypeCasts[0])
}

func TestInspectCompiledPackage_FilepathWalkFunctions(t *testing.T) {
	pkg, err := InspectCompiledPackage("path/filepath")
	require.NoError(t, err)
	require.NotNil(t, pkg)

	walkDir, ok := pkg.Funcs["walk_dir"]
	require.True(t, ok, "filepath.walk_dir should be bridged")
	require.Len(t, walkDir.Params, 2)
	assert.Equal(t, GoString, walkDir.Params[0])
	assert.Equal(t, GoFunc, walkDir.Params[1])
	require.Contains(t, walkDir.FuncTypes, 1)

	cb := walkDir.FuncTypes[1]
	require.NotNil(t, cb)
	assert.Equal(t, []GoType{GoString, GoAny, GoError}, cb.Params)
	assert.Equal(t, []GoType{GoError}, cb.Returns)
	require.Contains(t, cb.TypeCasts, 1)
	assert.Equal(t, "assert:fs.DirEntry", cb.TypeCasts[1])

	_, ok = pkg.Funcs["walk"]
	require.True(t, ok, "filepath.walk should be bridged")
}

func TestInspectSourcePackage_VariadicReadOptionSupport(t *testing.T) {
	result, err := InspectSourcePackage(fixtureDir("fixture_variadic_read"))
	require.NoError(t, err)

	FinalizeStructs(result, "varread", "varread")

	readSig, ok := result.Package.Funcs["read"]
	require.True(t, ok, "read should be bridged")
	require.True(t, readSig.Variadic, "read should stay variadic")
	require.Equal(t, []GoType{GoAny, GoAny}, readSig.Params)
	require.Contains(t, readSig.TypeCasts, 0)
	assert.Equal(t, "assert:interface{Read([]byte) (int, error)}", readSig.TypeCasts[0])
	require.Contains(t, readSig.TypeCasts, 1)
	assert.Equal(t, "varread.ReadOption", readSig.TypeCasts[1])

	var gpxInfo *GoStructInfo
	for i := range result.Package.Structs {
		if result.Package.Structs[i].GoName == "GPX" {
			gpxInfo = &result.Package.Structs[i]
			break
		}
	}
	require.NotNil(t, gpxInfo, "GPX struct should be discovered")

	fields := map[string]GoStructFieldInfo{}
	for _, f := range gpxInfo.Fields {
		fields[f.RugoName] = f
	}

	require.Contains(t, fields, "metadata")
	assert.Equal(t, "rugo_struct_varread_MetadataType", fields["metadata"].WrapType)
	assert.False(t, fields["metadata"].WrapValue)

	require.Contains(t, fields, "wpt")
	assert.Equal(t, "rugo_struct_varread_WptType", fields["wpt"].WrapSliceType)
	assert.False(t, fields["wpt"].WrapSliceElemValue)

	require.Contains(t, fields, "rte")
	assert.Equal(t, "rugo_struct_varread_RteType", fields["rte"].WrapSliceType)
	assert.False(t, fields["rte"].WrapSliceElemValue)

	require.Contains(t, fields, "trk")
	assert.Equal(t, "rugo_struct_varread_TrkType", fields["trk"].WrapSliceType)
	assert.False(t, fields["trk"].WrapSliceElemValue)
}

func TestFinalizeStructs_ModulePathPkgNameMismatch_DoesNotImportSelfByPkgName(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(`module example.com/go-hmod

go 1.22
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hmod.go"), []byte(`package hmod

type Node struct {
	Next  *Node
	Items []*Node
}

func NewNode() *Node { return &Node{} }
`), 0o644))

	result, err := InspectSourcePackage(dir)
	require.NoError(t, err)
	FinalizeStructs(result, "hmod", "hmod")

	require.Contains(t, result.Package.Funcs, "new_node")
	assert.NotContains(t, result.Package.ExtraImports, "hmod")
}

func TestExtractStructName_PrefersQualifiedExternalKey(t *testing.T) {
	pkg := types.NewPackage("example.com/gdk", "gdk")
	snapshot := types.NewNamed(types.NewTypeName(0, pkg, "Snapshot", nil), types.NewStruct(nil, nil), nil)
	known := map[string]bool{
		"Snapshot": true,
		ExternalTypeKey("example.com/gdk", "Snapshot"): true,
	}

	gotPtr := extractStructName(types.NewPointer(snapshot), known)
	gotVal := extractStructName(snapshot, known)
	want := ExternalTypeKey("example.com/gdk", "Snapshot")
	assert.Equal(t, want, gotPtr)
	assert.Equal(t, want, gotVal)
}

func TestCollectExternalFromType_DoesNotSkipOnNameCollision(t *testing.T) {
	pkg := types.NewPackage("example.com/gdk", "gdk")
	snapshot := types.NewNamed(types.NewTypeName(0, pkg, "Snapshot", nil), types.NewStruct(nil, nil), nil)
	known := map[string]bool{"Snapshot": true} // in-package collision name
	externals := map[string]ExternalTypeInfo{}

	collectExternalFromType(types.NewPointer(snapshot), "example.com/gtk", known, externals)

	key := ExternalTypeKey("example.com/gdk", "Snapshot")
	ext, ok := externals[key]
	require.True(t, ok, "external type should still be discovered despite name collision")
	assert.Equal(t, "gdk", ext.PkgName)
	assert.Equal(t, "Snapshot", ext.GoName)
}

func TestDiscoverEmbeddedFields_MarksValueVsPointer(t *testing.T) {
	pkg := types.NewPackage("example.com/test", "test")
	base := types.NewNamed(types.NewTypeName(0, pkg, "Base", nil), types.NewStruct(nil, nil), nil)
	node := types.NewNamed(types.NewTypeName(0, pkg, "Node", nil), types.NewStruct(nil, nil), nil)
	childStruct := types.NewStruct([]*types.Var{
		types.NewField(0, pkg, "Base", base, true),
		types.NewField(0, pkg, "Node", types.NewPointer(node), true),
	}, nil)
	child := types.NewNamed(types.NewTypeName(0, pkg, "Child", nil), childStruct, nil)

	fields := discoverEmbeddedFields(child, map[string]string{
		"Base": "rugo_struct_test_Base",
		"Node": "rugo_struct_test_Node",
	}, map[string]bool{
		"Base": true,
		"Node": true,
	})

	require.Len(t, fields, 2)
	byName := map[string]GoStructFieldInfo{}
	for _, f := range fields {
		byName[f.GoName] = f
	}
	require.Contains(t, byName, "Base")
	require.Contains(t, byName, "Node")
	assert.True(t, byName["Base"].WrapValue, "value embedded field must be marked WrapValue")
	assert.False(t, byName["Node"].WrapValue, "pointer embedded field must not be marked WrapValue")
}

func TestGenerateExternalOpaqueWrapper_UsesAddressForValueEmbedded(t *testing.T) {
	helper := GenerateExternalOpaqueWrapper("test", ExternalTypeInfo{
		PkgName: "gtk",
		GoName:  "Window",
		EmbeddedFields: []GoStructFieldInfo{
			{GoName: "Widget", RugoName: "widget", WrapType: "rugo_struct_test_Widget", WrapValue: true},
			{GoName: "Object", RugoName: "object", WrapType: "rugo_struct_test_Object", WrapValue: false},
		},
	})

	assert.Contains(t, helper.Code, "return interface{}(&rugo_struct_test_Widget{v: &w.v.Widget}), true")
	assert.Contains(t, helper.Code, "return interface{}(&rugo_struct_test_Object{v: w.v.Object}), true")
}

func TestFinalizeStructs_ConstructorHelperUsesRefreshedMethods(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/late\n\ngo 1.22\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "ext"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "ext", "ext.go"), []byte("package ext\n\ntype Snapshot struct{}\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(root, "local"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "local", "local.go"), []byte(`package local

import "example.com/late/ext"

type Snapshot struct{}
type Paintable struct{}

func (p *Paintable) Snapshot(s *ext.Snapshot, w, h float64) {}
`), 0o644))

	result, err := InspectSourcePackage(filepath.Join(root, "local"))
	require.NoError(t, err)
	FinalizeStructs(result, "local", "local")

	sig, ok := result.Package.Funcs["paintable"]
	require.True(t, ok, "paintable constructor should be registered")
	require.NotEmpty(t, sig.RuntimeHelpers, "constructor should carry struct wrapper helper")

	var helperCode string
	for _, h := range sig.RuntimeHelpers {
		if h.Key == "rugo_struct_local_Paintable" {
			helperCode = h.Code
			break
		}
	}
	require.NotEmpty(t, helperCode, "paintable constructor must include refreshed wrapper helper")
	assert.Contains(t, helperCode, "rugo_upcast_rugo_ext_local_ext_Snapshot(args[0]).v")
	assert.NotContains(t, helperCode, "rugo_upcast_rugo_struct_local_Snapshot(args[0]).v")
}

func TestNewExportPathResolver_CachesSuccess(t *testing.T) {
	orig := goListExportJSON
	t.Cleanup(func() { goListExportJSON = orig })

	calls := 0
	goListExportJSON = func(_, path string) ([]byte, error) {
		calls++
		return []byte(`{"Export":"/tmp/` + path + `.a"}`), nil
	}

	resolve := newExportPathResolver("/tmp")
	got1, err := resolve("example.com/mod")
	require.NoError(t, err)
	got2, err := resolve("example.com/mod")
	require.NoError(t, err)
	assert.Equal(t, got1, got2)
	assert.Equal(t, 1, calls, "resolver should cache successful lookups")
}

func TestNewExportPathResolver_CachesFailures(t *testing.T) {
	orig := goListExportJSON
	t.Cleanup(func() { goListExportJSON = orig })

	calls := 0
	goListExportJSON = func(_, _ string) ([]byte, error) {
		calls++
		return nil, errors.New("boom")
	}

	resolve := newExportPathResolver("/tmp")
	_, err := resolve("example.com/mod")
	require.Error(t, err)
	_, err = resolve("example.com/mod")
	require.Error(t, err)
	assert.Equal(t, 1, calls, "resolver should cache failed lookups too")
}

func TestModuleAwareImporter_FallsBackWhenGoListFails(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/fallback\n\ngo 1.22\n"), 0o644))

	orig := goListExportJSON
	t.Cleanup(func() { goListExportJSON = orig })
	goListExportJSON = func(_, _ string) ([]byte, error) {
		return nil, errors.New("go list timeout")
	}

	imp := moduleAwareImporter(token.NewFileSet(), root)
	pkg, err := imp.Import("fmt")
	require.NoError(t, err)
	assert.Equal(t, "fmt", pkg.Name())
}
