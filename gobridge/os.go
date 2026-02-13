package gobridge

func init() {
	Register(&Package{
		Path: "os",
		Doc:  "Operating system functions from Go's os package.",
		Funcs: map[string]GoFuncSig{
			"getenv":   {GoName: "Getenv", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Returns the value of the environment variable."},
			"setenv":   {GoName: "Setenv", Params: []GoType{GoString, GoString}, Returns: []GoType{GoError}, Doc: "Sets the value of the environment variable."},
			"unsetenv": {GoName: "Unsetenv", Params: []GoType{GoString}, Returns: []GoType{GoError}, Doc: "Removes the environment variable."},
			"hostname": {GoName: "Hostname", Params: nil, Returns: []GoType{GoString, GoError}, Doc: "Returns the host name reported by the kernel."},
			"getwd":    {GoName: "Getwd", Params: nil, Returns: []GoType{GoString, GoError}, Doc: "Returns the current working directory."},
			"mkdir_all": {
				GoName: "MkdirAll", Params: []GoType{GoString, GoInt}, Returns: []GoType{GoError},
				Doc:       "Creates a directory path and all parents that do not exist.",
				TypeCasts: map[int]string{1: "os.FileMode"},
			},
			"remove":     {GoName: "Remove", Params: []GoType{GoString}, Returns: []GoType{GoError}, Doc: "Removes the named file or empty directory."},
			"remove_all": {GoName: "RemoveAll", Params: []GoType{GoString}, Returns: []GoType{GoError}, Doc: "Removes path and any children it contains."},
			"read_file": {
				GoName: "ReadFile", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError},
				Doc: "Reads and returns the contents of the named file.",
			},
			"write_file": {
				GoName: "WriteFile", Params: []GoType{GoString, GoByteSlice, GoInt}, Returns: []GoType{GoError},
				Doc:       "Writes data to the named file with the given permissions.",
				TypeCasts: map[int]string{2: "os.FileMode"},
			},
			"temp_dir":      {GoName: "TempDir", Params: nil, Returns: []GoType{GoString}, Doc: "Returns the default directory for temporary files."},
			"user_home_dir": {GoName: "UserHomeDir", Params: nil, Returns: []GoType{GoString, GoError}, Doc: "Returns the current user's home directory."},
		},
	})
}
