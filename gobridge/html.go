package gobridge

func init() {
	Register(&Package{
		Path: "html",
		Doc:  "Functions from Go's html package.",
		Funcs: map[string]GoFuncSig{
			"escape_string": {GoName: "EscapeString", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"unescape_string": {GoName: "UnescapeString", Params: []GoType{GoString}, Returns: []GoType{GoString}},
		},
	})
}
