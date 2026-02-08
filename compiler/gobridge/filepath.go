package gobridge

func init() {
	Register(&Package{
		Path: "path/filepath",
		Funcs: map[string]GoFuncSig{
			"base":       {GoName: "Base", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"clean":      {GoName: "Clean", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"dir":        {GoName: "Dir", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"ext":        {GoName: "Ext", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"is_abs":     {GoName: "IsAbs", Params: []GoType{GoString}, Returns: []GoType{GoBool}},
			"join":       {GoName: "Join", Params: []GoType{GoStringSlice}, Returns: []GoType{GoString}, Variadic: true},
			"match":      {GoName: "Match", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool, GoError}},
			"rel":        {GoName: "Rel", Params: []GoType{GoString, GoString}, Returns: []GoType{GoString, GoError}},
			"split":      {GoName: "Split", Params: []GoType{GoString}, Returns: []GoType{GoString, GoString}},
			"to_slash":   {GoName: "ToSlash", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"from_slash": {GoName: "FromSlash", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"abs":        {GoName: "Abs", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}},
		},
	})
}
