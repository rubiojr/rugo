package gobridge

func init() {
	Register(&Package{
		Path: "math/rand/v2",
		Doc:  "Random number generation from Go's math/rand/v2 package.",
		Funcs: map[string]GoFuncSig{
			"int_n":   {GoName: "IntN", Params: []GoType{GoInt}, Returns: []GoType{GoInt}, Doc: "Returns a random int in [0, n)."},
			"float64": {GoName: "Float64", Params: nil, Returns: []GoType{GoFloat64}, Doc: "Returns a random float64 in [0.0, 1.0)."},
			"n":       {GoName: "IntN", Params: []GoType{GoInt}, Returns: []GoType{GoInt}, Doc: "Alias for int_n. Returns a random int in [0, n)."},
		},
	})
}
