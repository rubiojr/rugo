package gobridge

func init() {
	Register(&Package{
		Path: "strconv",
		Funcs: map[string]GoFuncSig{
			"atoi":         {GoName: "Atoi", Params: []GoType{GoString}, Returns: []GoType{GoInt, GoError}},
			"itoa":         {GoName: "Itoa", Params: []GoType{GoInt}, Returns: []GoType{GoString}},
			"format_bool":  {GoName: "FormatBool", Params: []GoType{GoBool}, Returns: []GoType{GoString}},
			"format_int":   {GoName: "FormatInt", Params: []GoType{GoInt, GoInt}, Returns: []GoType{GoString}},
			"format_float": {GoName: "FormatFloat", Params: []GoType{GoFloat64, GoByte, GoInt, GoInt}, Returns: []GoType{GoString}},
			"parse_bool":   {GoName: "ParseBool", Params: []GoType{GoString}, Returns: []GoType{GoBool, GoError}},
			"parse_int":    {GoName: "ParseInt", Params: []GoType{GoString, GoInt, GoInt}, Returns: []GoType{GoInt, GoError}},
			"parse_float":  {GoName: "ParseFloat", Params: []GoType{GoString, GoInt}, Returns: []GoType{GoFloat64, GoError}},
		},
	})
}
