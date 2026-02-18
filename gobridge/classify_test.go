package gobridge

import (
	"go/importer"
	"go/types"
	"testing"
)

func TestClassifyGoType_Basic(t *testing.T) {
	tests := []struct {
		kind     types.BasicKind
		wantType GoType
		wantTier Tier
	}{
		{types.String, GoString, TierAuto},
		{types.Int, GoInt, TierAuto},
		{types.Float64, GoFloat64, TierAuto},
		{types.Bool, GoBool, TierAuto},
		{types.Byte, GoByte, TierCastable},
		{types.Int64, GoInt64, TierCastable},
		{types.Float32, GoFloat32, TierCastable},
		{types.Uint, GoUint, TierCastable},
		{types.Uint32, GoUint32, TierCastable},
		{types.Uint64, GoUint64, TierCastable},
	}

	for _, tt := range tests {
		basic := types.Typ[tt.kind]
		gt, tier, _ := ClassifyGoType(basic, true)
		if gt != tt.wantType {
			t.Errorf("ClassifyGoType(%s): got type %d, want %d", basic.Name(), gt, tt.wantType)
		}
		if tier != tt.wantTier {
			t.Errorf("ClassifyGoType(%s): got tier %s, want %s", basic.Name(), tier, tt.wantTier)
		}
	}
}

func TestClassifyGoType_Blocked(t *testing.T) {
	// Pointer type
	ptrType := types.NewPointer(types.Typ[types.String])
	_, tier, _ := ClassifyGoType(ptrType, true)
	if tier != TierBlocked {
		t.Errorf("pointer type: got tier %s, want blocked", tier)
	}

	// Map type
	mapType := types.NewMap(types.Typ[types.String], types.Typ[types.Int])
	_, tier, _ = ClassifyGoType(mapType, true)
	if tier != TierBlocked {
		t.Errorf("map type: got tier %s, want blocked", tier)
	}

	// Chan type
	chanType := types.NewChan(types.SendRecv, types.Typ[types.Int])
	_, tier, _ = ClassifyGoType(chanType, true)
	if tier != TierBlocked {
		t.Errorf("chan type: got tier %s, want blocked", tier)
	}

	// Function pointer param should be treated as a callback.
	cbSig := types.NewSignatureType(nil, nil, nil, nil, nil, false)
	fnPtr := types.NewPointer(cbSig)
	gt, tier, _ := ClassifyGoType(fnPtr, true)
	if tier != TierFunc || gt != GoFunc {
		t.Errorf("*func() param: got type=%v tier=%s, want GoFunc/func", gt, tier)
	}
}

func TestClassifyGoType_Slice(t *testing.T) {
	strSlice := types.NewSlice(types.Typ[types.String])
	gt, tier, _ := ClassifyGoType(strSlice, true)
	if gt != GoStringSlice || tier != TierAuto {
		t.Errorf("[]string: got type=%d tier=%s, want GoStringSlice/auto", gt, tier)
	}

	byteSlice := types.NewSlice(types.Typ[types.Byte])
	gt, tier, _ = ClassifyGoType(byteSlice, true)
	if gt != GoByteSlice || tier != TierCastable {
		t.Errorf("[]byte: got type=%d tier=%s, want GoByteSlice/castable", gt, tier)
	}

	intSlice := types.NewSlice(types.Typ[types.Int])
	_, tier, _ = ClassifyGoType(intSlice, true)
	if tier != TierBlocked {
		t.Errorf("[]int: got tier %s, want blocked", tier)
	}
}

func TestClassifyFunc_StringsContains(t *testing.T) {
	imp := importer.Default()
	pkg, err := imp.Import("strings")
	if err != nil {
		t.Fatalf("importing strings: %v", err)
	}

	obj := pkg.Scope().Lookup("Contains")
	fn, ok := obj.(*types.Func)
	if !ok {
		t.Fatal("strings.Contains is not a Func")
	}

	sig := fn.Type().(*types.Signature)
	bf := ClassifyFunc("Contains", "contains", sig)

	if bf.Tier != TierAuto {
		t.Errorf("strings.Contains tier: got %s, want auto", bf.Tier)
	}
	if len(bf.Params) != 2 {
		t.Fatalf("strings.Contains params: got %d, want 2", len(bf.Params))
	}
	if bf.Params[0] != GoString || bf.Params[1] != GoString {
		t.Errorf("strings.Contains params: got %v, want [GoString, GoString]", bf.Params)
	}
	if len(bf.Returns) != 1 || bf.Returns[0] != GoBool {
		t.Errorf("strings.Contains returns: got %v, want [GoBool]", bf.Returns)
	}
}

func TestClassifyFunc_Blocked(t *testing.T) {
	// Build a function with a pointer param: func(p *string) string
	strPtr := types.NewPointer(types.Typ[types.String])
	params := types.NewTuple(types.NewVar(0, nil, "p", strPtr))
	results := types.NewTuple(types.NewVar(0, nil, "", types.Typ[types.String]))
	sig := types.NewSignatureType(nil, nil, nil, params, results, false)

	bf := ClassifyFunc("Blocked", "blocked", sig)
	if bf.Tier != TierBlocked {
		t.Errorf("func(*string) string: got tier %s, want blocked", bf.Tier)
	}
}

func TestClassifyFunc_FuncPointerParam(t *testing.T) {
	// Build func(cb *func(int))
	cbParams := types.NewTuple(types.NewVar(0, nil, "v", types.Typ[types.Int]))
	cbSig := types.NewSignatureType(nil, nil, nil, cbParams, nil, false)
	cbPtr := types.NewPointer(cbSig)
	params := types.NewTuple(types.NewVar(0, nil, "cb", cbPtr))
	sig := types.NewSignatureType(nil, nil, nil, params, nil, false)

	bf := ClassifyFunc("WithCb", "with_cb", sig)
	if bf.Tier == TierBlocked {
		t.Fatalf("func(*func(int)) should be bridgeable, got blocked: %s", bf.Reason)
	}
	if len(bf.Params) != 1 || bf.Params[0] != GoFunc {
		t.Fatalf("params = %v, want [GoFunc]", bf.Params)
	}
	if bf.FuncTypes == nil || bf.FuncTypes[0] == nil {
		t.Fatalf("FuncTypes[0] missing for func pointer param")
	}
	if bf.FuncParamPointer == nil || !bf.FuncParamPointer[0] {
		t.Fatalf("FuncParamPointer[0] should be true for *func param")
	}
}

func TestToSnakeCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Contains", "contains"},
		{"HasPrefix", "has_prefix"},
		{"ToUpper", "to_upper"},
		{"IsNaN", "is_nan"},
		{"FMA", "fma"},
		{"NewReader", "new_reader"},
		{"URLEncode", "url_encode"},
	}

	for _, tt := range tests {
		got := ToSnakeCase(tt.input)
		if got != tt.want {
			t.Errorf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGoTypeConst(t *testing.T) {
	tests := []struct {
		input GoType
		want  string
	}{
		{GoString, "GoString"},
		{GoInt, "GoInt"},
		{GoFloat64, "GoFloat64"},
		{GoBool, "GoBool"},
		{GoError, "GoError"},
		{GoFunc, "GoFunc"},
	}

	for _, tt := range tests {
		got := GoTypeConst(tt.input)
		if got != tt.want {
			t.Errorf("GoTypeConst(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTierString(t *testing.T) {
	tests := []struct {
		tier Tier
		want string
	}{
		{TierAuto, "auto"},
		{TierCastable, "castable"},
		{TierFunc, "func"},
		{TierBlocked, "blocked"},
	}
	for _, tt := range tests {
		if got := tt.tier.String(); got != tt.want {
			t.Errorf("Tier(%d).String() = %q, want %q", tt.tier, got, tt.want)
		}
	}
}

func TestTypeWrapReturn_GoIntCastsToInt(t *testing.T) {
	got := TypeWrapReturn("_v", GoInt)
	want := "interface{}(int(_v))"
	if got != want {
		t.Fatalf("TypeWrapReturn GoInt = %q, want %q", got, want)
	}
}
