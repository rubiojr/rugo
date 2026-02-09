package gobridge

func init() {
	Register(&Package{
		Path: "sort",
		Doc:  "Sorting functions from Go's sort package.",
		Funcs: map[string]GoFuncSig{
			"strings":   {GoName: "Strings", Params: []GoType{GoStringSlice}, Returns: nil, Doc: "Sorts a slice of strings in increasing order."},
			"ints":      {GoName: "Ints", Params: []GoType{GoStringSlice}, Returns: nil, Doc: "Sorts a slice of ints in increasing order."},
			"is_sorted": {GoName: "StringsAreSorted", Params: []GoType{GoStringSlice}, Returns: []GoType{GoBool}, Doc: "Reports whether a slice of strings is sorted."},
		},
	})
}
