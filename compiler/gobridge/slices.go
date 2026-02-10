package gobridge

import "fmt"

var slicesHelpers = []RuntimeHelper{
	{Key: "rugo_slices_contains", Code: `func rugo_slices_contains(v interface{}, target interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	ts := fmt.Sprintf("%v", target)
	for _, e := range arr {
		if fmt.Sprintf("%v", e) == ts { return interface{}(true) }
	}
	return interface{}(false)
}

`},
	{Key: "rugo_slices_index", Code: `func rugo_slices_index(v interface{}, target interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	ts := fmt.Sprintf("%v", target)
	for i, e := range arr {
		if fmt.Sprintf("%v", e) == ts { return interface{}(i) }
	}
	return interface{}(-1)
}

`},
	{Key: "rugo_slices_reverse", Code: `func rugo_slices_reverse(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	r := make([]interface{}, len(arr))
	for i, e := range arr { r[len(arr)-1-i] = e }
	return interface{}(r)
}

`},
	{Key: "rugo_slices_compact", Code: `func rugo_slices_compact(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	if len(arr) == 0 { return interface{}([]interface{}{}) }
	r := []interface{}{arr[0]}
	for i := 1; i < len(arr); i++ {
		if fmt.Sprintf("%v", arr[i]) != fmt.Sprintf("%v", arr[i-1]) {
			r = append(r, arr[i])
		}
	}
	return interface{}(r)
}

`},
}

func init() {
	Register(&Package{
		Path:       "slices",
		Doc:        "Array utility functions inspired by Go's slices package.",
		NoGoImport: true,
		Funcs: map[string]GoFuncSig{
			"contains": {
				GoName: "Contains", Params: []GoType{GoStringSlice, GoString}, Returns: []GoType{GoBool},
				Doc: "Reports whether a value is present in the array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_contains(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: slicesHelpers[:1],
			},
			"index": {
				GoName: "Index", Params: []GoType{GoStringSlice, GoString}, Returns: []GoType{GoInt},
				Doc: "Returns the index of the first occurrence of a value, or -1 if not found.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_index(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: slicesHelpers[1:2],
			},
			"reverse": {
				GoName: "Reverse", Params: []GoType{GoStringSlice}, Returns: []GoType{GoStringSlice},
				Doc: "Returns a new array with elements in reverse order.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_reverse(%s)", args[0])
				},
				RuntimeHelpers: slicesHelpers[2:3],
			},
			"compact": {
				GoName: "Compact", Params: []GoType{GoStringSlice}, Returns: []GoType{GoStringSlice},
				Doc: "Returns a new array with consecutive duplicate elements removed.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_compact(%s)", args[0])
				},
				RuntimeHelpers: slicesHelpers[3:4],
			},
		},
	})
}
