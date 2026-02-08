package gobridge

func init() {
	Register(&Package{
		Path: "sort",
		Funcs: map[string]GoFuncSig{
			"strings":   {GoName: "Strings", Params: []GoType{GoStringSlice}, Returns: nil},
			"ints":      {GoName: "Ints", Params: []GoType{GoStringSlice}, Returns: nil}, // special handling in codegen
			"is_sorted": {GoName: "StringsAreSorted", Params: []GoType{GoStringSlice}, Returns: []GoType{GoBool}},
		},
	})
}
