package gobridge

func init() {
	Register(&Package{
		Path: "time",
		Funcs: map[string]GoFuncSig{
			"now_unix":      {GoName: "Now().Unix", Params: nil, Returns: []GoType{GoInt}},
			"now_unix_nano": {GoName: "Now().UnixNano", Params: nil, Returns: []GoType{GoInt}},
			"sleep_ms":      {GoName: "Sleep", Params: []GoType{GoInt}, Returns: nil},
		},
	})
}
