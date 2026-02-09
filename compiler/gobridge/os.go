package gobridge

func init() {
	Register(&Package{
		Path: "os",
		Funcs: map[string]GoFuncSig{
			"getenv":        {GoName: "Getenv", Params: []GoType{GoString}, Returns: []GoType{GoString}},
			"setenv":        {GoName: "Setenv", Params: []GoType{GoString, GoString}, Returns: []GoType{GoError}},
			"unsetenv":      {GoName: "Unsetenv", Params: []GoType{GoString}, Returns: []GoType{GoError}},
			"hostname":      {GoName: "Hostname", Params: nil, Returns: []GoType{GoString, GoError}},
			"getwd":         {GoName: "Getwd", Params: nil, Returns: []GoType{GoString, GoError}},
			"mkdir_all":     {GoName: "MkdirAll", Params: []GoType{GoString, GoInt}, Returns: []GoType{GoError}},
			"remove":        {GoName: "Remove", Params: []GoType{GoString}, Returns: []GoType{GoError}},
			"remove_all":    {GoName: "RemoveAll", Params: []GoType{GoString}, Returns: []GoType{GoError}},
			"read_file":     {GoName: "ReadFile", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}},
			"write_file":    {GoName: "WriteFile", Params: []GoType{GoString, GoString, GoInt}, Returns: []GoType{GoError}},
			"temp_dir":      {GoName: "TempDir", Params: nil, Returns: []GoType{GoString}},
			"user_home_dir": {GoName: "UserHomeDir", Params: nil, Returns: []GoType{GoString, GoError}},
		},
	})
}
