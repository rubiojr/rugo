package gobridge

import "fmt"

var mapsHelpers = []RuntimeHelper{helperFromFile("rugo_maps_helpers", mapsHelperSrc)}

func init() {
	Register(&Package{
		Path:         "maps",
		Doc:          "Hash/map utility functions inspired by Go's maps package.",
		NoGoImport:   true,
		ExtraImports: []string{"sort"},
		Funcs: map[string]GoFuncSig{
			"keys": {
				GoName: "Keys", Params: []GoType{GoString}, Returns: []GoType{GoStringSlice},
				Doc: "Returns the sorted keys of a hash as an array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_keys(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers,
			},
			"values": {
				GoName: "Values", Params: []GoType{GoString}, Returns: []GoType{GoStringSlice},
				Doc: "Returns the values of a hash as an array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_values(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers,
			},
			"clone": {
				GoName: "Clone", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Returns a shallow copy of a hash.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_clone(%s)", args[0])
				},
				RuntimeHelpers: mapsHelpers,
			},
			"equal": {
				GoName: "Equal", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool},
				Doc: "Reports whether two hashes have the same keys and values.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_maps_equal(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: mapsHelpers,
			},
		},
	})
}
