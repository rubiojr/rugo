package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "path/filepath",
		Doc:  "File path manipulation functions from Go's path/filepath package.",
		Funcs: map[string]GoFuncSig{
			"base":       {GoName: "Base", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Returns the last element of path."},
			"clean":      {GoName: "Clean", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Returns the shortest path name equivalent to path."},
			"dir":        {GoName: "Dir", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Returns all but the last element of path."},
			"ext":        {GoName: "Ext", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Returns the file name extension used by path."},
			"is_abs":     {GoName: "IsAbs", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the path is absolute."},
			"join":       {GoName: "Join", Params: []GoType{GoStringSlice}, Returns: []GoType{GoString}, Variadic: true, Doc: "Joins path elements into a single path."},
			"match":      {GoName: "Match", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool, GoError}, Doc: "Reports whether name matches the shell pattern."},
			"rel":        {GoName: "Rel", Params: []GoType{GoString, GoString}, Returns: []GoType{GoString, GoError}, Doc: "Returns a relative path from basepath to targpath."},
			"to_slash":   {GoName: "ToSlash", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Replaces OS path separators with slashes."},
			"from_slash": {GoName: "FromSlash", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Replaces slashes with OS path separators."},
			"abs":        {GoName: "Abs", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}, Doc: "Returns an absolute representation of path."},
			"split": {
				GoName: "Split", Params: []GoType{GoString}, Returns: []GoType{GoString, GoString},
				Doc: "Splits path into directory and file components.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("func() interface{} { _d, _f := %s.Split(%s); return interface{}([]interface{}{interface{}(_d), interface{}(_f)}) }()",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
		},
	})
}
