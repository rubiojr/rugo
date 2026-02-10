package gobridge

import "fmt"

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

var intSliceHelpers = []RuntimeHelper{
	{Key: "rugo_go_to_int_slice", Code: `func rugo_go_to_int_slice(v interface{}) []int {
	arr, ok := v.([]interface{})
	if !ok { panic(fmt.Sprintf("expected array, got %T", v)) }
	r := make([]int, len(arr))
	for i, s := range arr { r[i] = rugo_to_int(s) }
	return r
}

func rugo_go_from_int_slice(v []int) interface{} {
	r := make([]interface{}, len(v))
	for i, s := range v { r[i] = interface{}(s) }
	return interface{}(r)
}

`},
}

func init() {
	Register(&Package{
		Path: "sort",
		Doc:  "Sorting functions from Go's sort package.",
		Funcs: map[string]GoFuncSig{
			"strings": {
				GoName: "Strings", Params: []GoType{GoStringSlice}, Returns: nil,
				Doc: "Sorts a slice of strings in increasing order.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("func() interface{} { _s := rugo_go_to_string_slice(%s); %s.Strings(_s); return rugo_go_from_string_slice(_s) }()", args[0], pkgBase)
				},
				RuntimeHelpers: stringSliceHelpers,
			},
			"ints": {
				GoName: "Ints", Params: []GoType{GoStringSlice}, Returns: nil,
				Doc: "Sorts a slice of ints in increasing order.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("func() interface{} { _s := rugo_go_to_int_slice(%s); %s.Ints(_s); return rugo_go_from_int_slice(_s) }()", args[0], pkgBase)
				},
				RuntimeHelpers: intSliceHelpers,
			},
			"is_sorted": {
				GoName: "StringsAreSorted", Params: []GoType{GoStringSlice}, Returns: []GoType{GoBool},
				Doc: "Reports whether a slice of strings is sorted.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("interface{}(%s.StringsAreSorted(rugo_go_to_string_slice(%s)))", pkgBase, args[0])
				},
				RuntimeHelpers: stringSliceHelpers,
			},
		},
	})
}
