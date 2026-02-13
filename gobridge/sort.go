package gobridge

var stringSliceHelpers = []RuntimeHelper{
	{Key: "rugo_go_to_string_slice", Code: `func rugo_go_to_string_slice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	r := make([]string, len(arr))
	for i, s := range arr { r[i] = rugo_to_string(s) }
	return r
}

func rugo_go_from_string_slice(v []string) interface{} {
	r := make([]interface{}, len(v))
	for i, s := range v { r[i] = interface{}(s) }
	return interface{}(r)
}

`},
}

var sortHelpers = []RuntimeHelper{
	{Key: "rugo_sort_in_place", Code: `func rugo_sort_in_place(v interface{}) {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	sort.Slice(arr, func(i, j int) bool {
		return rugo_compare(arr[i], arr[j]) < 0
	})
}

func rugo_sort_is_sorted(v interface{}) bool {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	return sort.SliceIsSorted(arr, func(i, j int) bool {
		return rugo_compare(arr[i], arr[j]) < 0
	})
}

`},
}

func init() {
	Register(&Package{
		Path:         "sort",
		Doc:          "Sorting functions from Go's sort package.",
		NoGoImport:   true,
		ExtraImports: []string{"sort"},
		Funcs: map[string]GoFuncSig{
			"strings": {
				GoName: "Strings", Params: []GoType{GoStringSlice}, Returns: nil,
				Doc: "Sorts an array of strings in increasing order (in-place).",
				Codegen: func(_ string, args []string, _ string) string {
					return "func() interface{} { rugo_sort_in_place(" + args[0] + "); return nil }()"
				},
				RuntimeHelpers: sortHelpers,
			},
			"ints": {
				GoName: "Ints", Params: []GoType{GoStringSlice}, Returns: nil,
				Doc: "Sorts an array of ints in increasing order (in-place).",
				Codegen: func(_ string, args []string, _ string) string {
					return "func() interface{} { rugo_sort_in_place(" + args[0] + "); return nil }()"
				},
				RuntimeHelpers: sortHelpers,
			},
			"is_sorted": {
				GoName: "IsSorted", Params: []GoType{GoStringSlice}, Returns: []GoType{GoBool},
				Doc: "Reports whether an array is sorted in increasing order.",
				Codegen: func(_ string, args []string, _ string) string {
					return "interface{}(rugo_sort_is_sorted(" + args[0] + "))"
				},
				RuntimeHelpers: sortHelpers,
			},
		},
	})
}
