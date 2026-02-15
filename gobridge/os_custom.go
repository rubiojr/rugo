package gobridge

func init() {
	// Functions needing TypeCasts for os.FileMode (not auto-generable).
	Extend("os", map[string]GoFuncSig{
		"args": {
			GoName: "Args", Params: nil, Returns: []GoType{GoStringSlice},
			Doc: "Returns the command-line arguments, starting with the program name.",
			Codegen: func(pkgBase string, args []string, rugoName string) string {
				return "rugo_go_from_string_slice(os.Args)"
			},
			RuntimeHelpers: []RuntimeHelper{StringSliceHelper},
		},
		"mkdir_all": {
			GoName: "MkdirAll", Params: []GoType{GoString, GoInt}, Returns: []GoType{GoError},
			Doc:       "Creates a directory path and all parents that do not exist.",
			TypeCasts: map[int]string{1: "os.FileMode"},
		},
		"write_file": {
			GoName: "WriteFile", Params: []GoType{GoString, GoByteSlice, GoInt}, Returns: []GoType{GoError},
			Doc:       "Writes data to the named file with the given permissions.",
			TypeCasts: map[int]string{2: "os.FileMode"},
		},
	})
}
