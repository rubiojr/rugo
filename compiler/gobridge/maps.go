package gobridge

import "fmt"

var mapsHelpers = []RuntimeHelper{
	{Key: "rugo_maps_keys", Code: `func rugo_maps_keys(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok { panic(fmt.Sprintf("expected hash, got %T", v)) }
	keys := make([]string, 0, len(m))
	for k := range m { keys = append(keys, fmt.Sprintf("%v", k)) }
	sort.Strings(keys)
	r := make([]interface{}, len(keys))
	for i, k := range keys { r[i] = interface{}(k) }
	return interface{}(r)
}

`},
	{Key: "rugo_maps_values", Code: `func rugo_maps_values(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok { panic(fmt.Sprintf("expected hash, got %T", v)) }
	keys := make([]string, 0, len(m))
	keyMap := make(map[string]interface{})
	for k := range m { s := fmt.Sprintf("%v", k); keys = append(keys, s); keyMap[s] = k }
	sort.Strings(keys)
	r := make([]interface{}, len(keys))
	for i, k := range keys { r[i] = m[keyMap[k]] }
	return interface{}(r)
}

`},
	{Key: "rugo_maps_clone", Code: `func rugo_maps_clone(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok { panic(fmt.Sprintf("expected hash, got %T", v)) }
	r := make(map[interface{}]interface{}, len(m))
	for k, val := range m { r[k] = val }
	return interface{}(r)
}

`},
	{Key: "rugo_maps_equal", Code: `func rugo_maps_equal(a interface{}, b interface{}) interface{} {
	ma, ok1 := a.(map[interface{}]interface{})
	mb, ok2 := b.(map[interface{}]interface{})
	if !ok1 || !ok2 { return interface{}(false) }
	if len(ma) != len(mb) { return interface{}(false) }
	for k, va := range ma {
		vb, ok := mb[k]
		if !ok { return interface{}(false) }
		if fmt.Sprintf("%v", va) != fmt.Sprintf("%v", vb) { return interface{}(false) }
	}
	return interface{}(true)
}

`},
}

func init() {
	Register(&Package{
		Path:         "maps",
		Doc:          "Hash/map utility functions inspired by Go's maps package.",
		NoGoImport:   true,
		ExtraImports: []string{"sort"},
		Funcs: map[string]GoFuncSig{
			"keys": {
				GoName: "Keys", Params: []GoType{GoString}, Returns: []GoType{GoStringSlice},
				Doc: "Returns the sorted keys of a hash as an array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_keys(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers[:1],
			},
			"values": {
				GoName: "Values", Params: []GoType{GoString}, Returns: []GoType{GoStringSlice},
				Doc: "Returns the values of a hash as an array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_values(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers[1:2],
			},
			"clone": {
				GoName: "Clone", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Returns a shallow copy of a hash.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_clone(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers[2:3],
			},
			"equal": {
				GoName: "Equal", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool},
				Doc: "Reports whether two hashes have the same keys and values.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_equal(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: mapsHelpers[3:4],
			},
		},
	})
}
