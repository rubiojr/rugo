package gobridge

import (
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ClassifyFuncType tests ---

func TestClassifyFuncType_PrimitiveCallback(t *testing.T) {
	// func(row int) — should always work, even without struct context.
	params := types.NewTuple(types.NewVar(0, nil, "row", types.Typ[types.Int]))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	ft := ClassifyFuncType(sig, nil, nil)
	require.NotNil(t, ft, "primitive callback should be bridgeable")
	assert.Equal(t, []GoType{GoInt}, ft.Params)
	assert.Nil(t, ft.StructCasts)
}

func TestClassifyFuncType_NoArgsCallback(t *testing.T) {
	// func() — no params, no returns.
	sig := types.NewSignatureType(nil, nil, nil, nil, nil, false)

	ft := ClassifyFuncType(sig, nil, nil)
	require.NotNil(t, ft)
	assert.Empty(t, ft.Params)
}

func TestClassifyFuncType_StructPointerParam_WithoutContext(t *testing.T) {
	// func(item *MyStruct) — blocked without struct context.
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, nil, "MyStruct", nil), structType, nil)
	ptrType := types.NewPointer(named)
	params := types.NewTuple(types.NewVar(0, nil, "item", ptrType))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	ft := ClassifyFuncType(sig, nil, nil)
	assert.Nil(t, ft, "struct pointer param without struct context should be blocked")
}

func TestClassifyFuncType_StructPointerParam_WithContext(t *testing.T) {
	// func(item *MyStruct) — bridgeable when struct context is available.
	pkg := types.NewPackage("example.com/test", "test")
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "MyStruct", nil), structType, nil)
	ptrType := types.NewPointer(named)
	params := types.NewTuple(types.NewVar(0, nil, "item", ptrType))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{"MyStruct": "rugo_struct_test_MyStruct"}
	known := map[string]bool{"MyStruct": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	require.NotNil(t, ft, "struct pointer param with context should be bridgeable")
	assert.Equal(t, "rugo_struct_test_MyStruct", ft.StructCasts[0])
	assert.Equal(t, "test.MyStruct", ft.StructGoTypes[0])
	assert.Nil(t, ft.StructParamValue, "pointer param should NOT be marked as value type")
}

func TestClassifyFuncType_StructValueParam(t *testing.T) {
	// func(d Date) — value type struct in callback.
	pkg := types.NewPackage("example.com/test", "test")
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "Date", nil), structType, nil)
	// Value type, not pointer.
	params := types.NewTuple(types.NewVar(0, nil, "d", named))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{"Date": "rugo_struct_test_Date"}
	known := map[string]bool{"Date": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	require.NotNil(t, ft, "struct value param with context should be bridgeable")
	assert.Equal(t, "rugo_struct_test_Date", ft.StructCasts[0])
	assert.Equal(t, "test.Date", ft.StructGoTypes[0])
	assert.True(t, ft.StructParamValue[0], "value type param should be marked")
}

func TestClassifyFuncType_MixedStructAndPrimitive(t *testing.T) {
	// func(item *MyStruct, column int)
	pkg := types.NewPackage("example.com/test", "test")
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "Item", nil), structType, nil)
	ptrType := types.NewPointer(named)
	params := types.NewTuple(
		types.NewVar(0, nil, "item", ptrType),
		types.NewVar(0, nil, "column", types.Typ[types.Int]),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{"Item": "rugo_struct_test_Item"}
	known := map[string]bool{"Item": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	require.NotNil(t, ft)
	assert.Equal(t, "rugo_struct_test_Item", ft.StructCasts[0])
	assert.Equal(t, GoInt, ft.Params[1])
	_, hasStructCast1 := ft.StructCasts[1]
	assert.False(t, hasStructCast1, "int param should have no struct cast")
}

func TestClassifyFuncType_MultipleStructParams(t *testing.T) {
	// func(a *Item, b *Item)
	pkg := types.NewPackage("example.com/test", "test")
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "Item", nil), structType, nil)
	ptrType := types.NewPointer(named)
	params := types.NewTuple(
		types.NewVar(0, nil, "a", ptrType),
		types.NewVar(0, nil, "b", ptrType),
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{"Item": "rugo_struct_test_Item"}
	known := map[string]bool{"Item": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	require.NotNil(t, ft)
	assert.Equal(t, "rugo_struct_test_Item", ft.StructCasts[0])
	assert.Equal(t, "rugo_struct_test_Item", ft.StructCasts[1])
}

func TestClassifyFuncType_ValueAndPointerMixed(t *testing.T) {
	// func(d Date, item *Item)
	pkg := types.NewPackage("example.com/test", "test")
	st := types.NewStruct(nil, nil)
	dateNamed := types.NewNamed(types.NewTypeName(0, pkg, "Date", nil), st, nil)
	itemNamed := types.NewNamed(types.NewTypeName(0, pkg, "Item", nil), st, nil)
	params := types.NewTuple(
		types.NewVar(0, nil, "d", dateNamed),         // value type
		types.NewVar(0, nil, "item", types.NewPointer(itemNamed)), // pointer type
	)
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{
		"Date": "rugo_struct_test_Date",
		"Item": "rugo_struct_test_Item",
	}
	known := map[string]bool{"Date": true, "Item": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	require.NotNil(t, ft)
	assert.True(t, ft.StructParamValue[0], "Date should be value type")
	assert.False(t, ft.StructParamValue[1], "Item should NOT be value type (pointer)")
	assert.Equal(t, "test.Date", ft.StructGoTypes[0])
	assert.Equal(t, "test.Item", ft.StructGoTypes[1])
}

func TestClassifyFuncType_UnknownStructStillBlocked(t *testing.T) {
	// func(item *Unknown) — struct not in known set.
	pkg := types.NewPackage("example.com/test", "test")
	structType := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "Unknown", nil), structType, nil)
	ptrType := types.NewPointer(named)
	params := types.NewTuple(types.NewVar(0, nil, "item", ptrType))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	wrappers := map[string]string{"Item": "rugo_struct_test_Item"}
	known := map[string]bool{"Item": true}

	ft := ClassifyFuncType(sig, wrappers, known)
	assert.Nil(t, ft, "unknown struct should remain blocked")
}

func TestClassifyFuncType_ChanParamStillBlocked(t *testing.T) {
	// func(ch chan int) — chan is always blocked.
	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	params := types.NewTuple(types.NewVar(0, nil, "ch", chanType))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	ft := ClassifyFuncType(sig, map[string]string{}, map[string]bool{})
	assert.Nil(t, ft, "chan param should remain blocked")
}

func TestClassifyFuncType_NestedFuncParamStillBlocked(t *testing.T) {
	// func(fn func()) — func-in-func is blocked.
	innerSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	params := types.NewTuple(types.NewVar(0, nil, "fn", innerSig))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	ft := ClassifyFuncType(sig, map[string]string{}, map[string]bool{})
	assert.Nil(t, ft, "nested func param should remain blocked")
}

// --- FuncAdapterConv tests ---

func TestFuncAdapterConv_PrimitiveOnly(t *testing.T) {
	ft := &GoFuncType{Params: []GoType{GoInt}}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "func(_p0 int)")
	assert.Contains(t, result, "interface{}(_p0)")
}

func TestFuncAdapterConv_PointerStruct(t *testing.T) {
	ft := &GoFuncType{
		Params:        []GoType{GoString}, // placeholder
		StructCasts:   map[int]string{0: "rugo_struct_qt6_QListWidgetItem"},
		StructGoTypes: map[int]string{0: "qt6.QListWidgetItem"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 *qt6.QListWidgetItem", "pointer struct should use *Type")
	assert.Contains(t, result, "&rugo_struct_qt6_QListWidgetItem{v: _p0}", "should wrap without &")
}

func TestFuncAdapterConv_ValueStruct(t *testing.T) {
	ft := &GoFuncType{
		Params:           []GoType{GoString},
		StructCasts:      map[int]string{0: "rugo_struct_qt6_QDate"},
		StructParamValue: map[int]bool{0: true},
		StructGoTypes:    map[int]string{0: "qt6.QDate"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 qt6.QDate", "value struct should NOT have *")
	assert.Contains(t, result, "&rugo_struct_qt6_QDate{v: &_p0}", "should take address with &")
}

func TestFuncAdapterConv_MixedStructAndPrimitive(t *testing.T) {
	ft := &GoFuncType{
		Params:        []GoType{GoString, GoInt},
		StructCasts:   map[int]string{0: "rugo_struct_qt6_QTreeWidgetItem"},
		StructGoTypes: map[int]string{0: "qt6.QTreeWidgetItem"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 *qt6.QTreeWidgetItem")
	assert.Contains(t, result, "_p1 int")
	assert.Contains(t, result, "interface{}(_p1)")
}

func TestFuncAdapterConv_NestedTypeName(t *testing.T) {
	// Double underscores in Go names (e.g., QAbstractTextDocumentLayout__PaintContext)
	ft := &GoFuncType{
		Params:        []GoType{GoString},
		StructCasts:   map[int]string{0: "rugo_struct_qt6_QAbstractTextDocumentLayout__PaintContext"},
		StructGoTypes: map[int]string{0: "qt6.QAbstractTextDocumentLayout__PaintContext"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 *qt6.QAbstractTextDocumentLayout__PaintContext")
	assert.NotContains(t, result, "qt6_QAbstractTextDocumentLayout_.", "must not incorrectly split on underscore")
}

func TestFuncAdapterConv_VoidCallback(t *testing.T) {
	ft := &GoFuncType{Params: []GoType{GoInt}}
	result := FuncAdapterConv("_arg", ft)
	// Void callbacks should not have `return`
	assert.NotContains(t, result, "return ")
	assert.Contains(t, result, "func(_p0 int)")
}

func TestFuncAdapterConv_WithReturn(t *testing.T) {
	ft := &GoFuncType{
		Params:  []GoType{GoString},
		Returns: []GoType{GoBool},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "return ")
	assert.Contains(t, result, " bool")
}

// --- qualifiedGoTypeName tests ---

func TestQualifiedGoTypeName_PointerToNamed(t *testing.T) {
	pkg := types.NewPackage("example.com/qt6", "qt6")
	st := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "QListWidgetItem", nil), st, nil)
	ptr := types.NewPointer(named)

	assert.Equal(t, "qt6.QListWidgetItem", qualifiedGoTypeName(ptr))
}

func TestQualifiedGoTypeName_ValueNamed(t *testing.T) {
	pkg := types.NewPackage("example.com/qt6", "qt6")
	st := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "QDate", nil), st, nil)

	assert.Equal(t, "qt6.QDate", qualifiedGoTypeName(named))
}

func TestQualifiedGoTypeName_NestedType(t *testing.T) {
	pkg := types.NewPackage("example.com/qt6", "qt6")
	st := types.NewStruct(nil, nil)
	named := types.NewNamed(types.NewTypeName(0, pkg, "QAbstractTextDocumentLayout__PaintContext", nil), st, nil)
	ptr := types.NewPointer(named)

	assert.Equal(t, "qt6.QAbstractTextDocumentLayout__PaintContext", qualifiedGoTypeName(ptr))
}

// --- Integration test with fixture_func_struct ---

func TestFuncStructCallbacks_Integration(t *testing.T) {
	dir := fixtureDir("fixture_func_struct")
	result, err := InspectSourcePackage(dir)
	require.NoError(t, err)

	FinalizeStructs(result, "funcstruct", "funcstruct")

	// Find Widget struct
	var widget *GoStructInfo
	for i := range result.Package.Structs {
		if result.Package.Structs[i].GoName == "Widget" {
			widget = &result.Package.Structs[i]
			break
		}
	}
	require.NotNil(t, widget, "Widget struct must be found")

	findMethod := func(name string) *GoStructMethodInfo {
		for i := range widget.Methods {
			if widget.Methods[i].GoName == name {
				return &widget.Methods[i]
			}
		}
		return nil
	}

	// OnPrimitiveOnly — func(row int): always works
	t.Run("OnPrimitiveOnly", func(t *testing.T) {
		m := findMethod("OnPrimitiveOnly")
		require.NotNil(t, m, "OnPrimitiveOnly must be bridged")
		assert.Equal(t, GoFunc, m.Params[0])
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Equal(t, []GoType{GoInt}, ft.Params)
		assert.Nil(t, ft.StructCasts)
	})

	// OnNoArgs — func(): always works
	t.Run("OnNoArgs", func(t *testing.T) {
		m := findMethod("OnNoArgs")
		require.NotNil(t, m, "OnNoArgs must be bridged")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Empty(t, ft.Params)
	})

	// OnItemClicked — func(item *Item): pointer struct param
	t.Run("OnItemClicked", func(t *testing.T) {
		m := findMethod("OnItemClicked")
		require.NotNil(t, m, "OnItemClicked must be bridged (struct callback)")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Contains(t, ft.StructCasts, 0)
		assert.Equal(t, "funcstruct.Item", ft.StructGoTypes[0])
		assert.False(t, ft.StructParamValue[0], "pointer param should not be value type")
	})

	// OnDateChanged — func(date Date): value struct param
	t.Run("OnDateChanged", func(t *testing.T) {
		m := findMethod("OnDateChanged")
		require.NotNil(t, m, "OnDateChanged must be bridged (value struct callback)")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Contains(t, ft.StructCasts, 0)
		assert.Equal(t, "funcstruct.Date", ft.StructGoTypes[0])
		assert.True(t, ft.StructParamValue[0], "value type must be marked")
	})

	// OnMixed — func(item *Item, column int): struct + primitive
	t.Run("OnMixed", func(t *testing.T) {
		m := findMethod("OnMixed")
		require.NotNil(t, m, "OnMixed must be bridged")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Contains(t, ft.StructCasts, 0)
		assert.Equal(t, GoInt, ft.Params[1])
		assert.NotContains(t, ft.StructCasts, 1)
	})

	// OnMultiStruct — func(a *Item, b *Item): multiple struct params
	t.Run("OnMultiStruct", func(t *testing.T) {
		m := findMethod("OnMultiStruct")
		require.NotNil(t, m, "OnMultiStruct must be bridged")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Contains(t, ft.StructCasts, 0)
		assert.Contains(t, ft.StructCasts, 1)
	})

	// OnValueAndPointer — func(d Date, item *Item): mixed value/pointer
	t.Run("OnValueAndPointer", func(t *testing.T) {
		m := findMethod("OnValueAndPointer")
		require.NotNil(t, m, "OnValueAndPointer must be bridged")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.True(t, ft.StructParamValue[0], "Date should be value type")
		assert.False(t, ft.StructParamValue[1], "Item should be pointer type")
	})

	// OnNestedType — func(ctx *Nested__Type): double-underscore name
	t.Run("OnNestedType", func(t *testing.T) {
		m := findMethod("OnNestedType")
		require.NotNil(t, m, "OnNestedType must be bridged")
		ft := m.FuncTypes[0]
		require.NotNil(t, ft)
		assert.Equal(t, "funcstruct.Nested__Type", ft.StructGoTypes[0])
		assert.False(t, ft.StructParamValue[0])
	})

	// OnChanCallback — func(ch chan int): must remain blocked
	t.Run("OnChanCallback_blocked", func(t *testing.T) {
		m := findMethod("OnChanCallback")
		assert.Nil(t, m, "OnChanCallback should be blocked (chan in callback)")
	})

	// OnFuncCallback — func(fn func()): must remain blocked
	t.Run("OnFuncCallback_blocked", func(t *testing.T) {
		m := findMethod("OnFuncCallback")
		assert.Nil(t, m, "OnFuncCallback should be blocked (func-in-func)")
	})

	// SetName — regular method, should still work
	t.Run("SetName_regular", func(t *testing.T) {
		m := findMethod("SetName")
		require.NotNil(t, m)
		assert.Equal(t, []GoType{GoString}, m.Params)
	})
}

// --- FuncAdapterConv integration: verify generated code compiles conceptually ---

func TestFuncAdapterConv_MultipleStructParams(t *testing.T) {
	ft := &GoFuncType{
		Params:        []GoType{GoString, GoString},
		StructCasts:   map[int]string{0: "rugo_struct_test_Item", 1: "rugo_struct_test_Item"},
		StructGoTypes: map[int]string{0: "test.Item", 1: "test.Item"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 *test.Item")
	assert.Contains(t, result, "_p1 *test.Item")
	// Both should be wrapped
	assert.Equal(t, 2, strings.Count(result, "rugo_struct_test_Item{v:"))
}

func TestFuncAdapterConv_ValueAndPointerMixed(t *testing.T) {
	ft := &GoFuncType{
		Params:           []GoType{GoString, GoString},
		StructCasts:      map[int]string{0: "rugo_struct_qt6_QDate", 1: "rugo_struct_qt6_QItem"},
		StructParamValue: map[int]bool{0: true},
		StructGoTypes:    map[int]string{0: "qt6.QDate", 1: "qt6.QItem"},
	}
	result := FuncAdapterConv("_arg", ft)
	assert.Contains(t, result, "_p0 qt6.QDate", "value type: no pointer")
	assert.Contains(t, result, "_p1 *qt6.QItem", "pointer type: has pointer")
	assert.Contains(t, result, "{v: &_p0}", "value type: takes address")
	assert.Contains(t, result, "{v: _p1}", "pointer type: no extra &")
	assert.NotContains(t, result, "{v: &_p1}")
}
