package gobridge

var sortHelpers = []RuntimeHelper{helperFromFile("rugo_sort_in_place", sortHelperSrc)}

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
