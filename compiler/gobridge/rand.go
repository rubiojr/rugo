package gobridge

func init() {
	Register(&Package{
		Path: "math/rand/v2",
		Funcs: map[string]GoFuncSig{
			"int_n": {GoName: "IntN", Params: []GoType{GoInt}, Returns: []GoType{GoInt}},
			"float64": {GoName: "Float64", Params: nil, Returns: []GoType{GoFloat64}},
			"n": {GoName: "IntN", Params: []GoType{GoInt}, Returns: []GoType{GoInt}},
		},
	})
}
