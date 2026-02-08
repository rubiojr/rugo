package gobridge

func init() {
	Register(&Package{
		Path: "regexp",
		Funcs: map[string]GoFuncSig{
			"match_string": {GoName: "MatchString", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool, GoError}},
		},
	})
}
