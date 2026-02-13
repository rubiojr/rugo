package gobridge

import "fmt"

var slicesHelpers = []RuntimeHelper{helperFromFile("rugo_slices_helpers", slicesHelperSrc)}

func init() {
	Register(&Package{
		Path:       "slices",
		Doc:        "Array utility functions inspired by Go's slices package.",
		NoGoImport: true,
		Funcs: map[string]GoFuncSig{
			"contains": {
				GoName: "Contains", Params: []GoType{GoStringSlice, GoString}, Returns: []GoType{GoBool},
				Doc: "Reports whether a value is present in the array.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_contains(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: slicesHelpers,
			},
			"index": {
				GoName: "Index", Params: []GoType{GoStringSlice, GoString}, Returns: []GoType{GoInt},
				Doc: "Returns the index of the first occurrence of a value, or -1 if not found.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_index(%s, %s)", args[0], args[1])
				},
				RuntimeHelpers: slicesHelpers,
			},
			"reverse": {
				GoName: "Reverse", Params: []GoType{GoStringSlice}, Returns: []GoType{GoStringSlice},
				Doc: "Returns a new array with elements in reverse order.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_reverse(%s)", args[0])
				},
				RuntimeHelpers: slicesHelpers,
			},
			"compact": {
				GoName: "Compact", Params: []GoType{GoStringSlice}, Returns: []GoType{GoStringSlice},
				Doc: "Returns a new array with consecutive duplicate elements removed.",
				Codegen: func(_ string, args []string, _ string) string {
					return fmt.Sprintf("rugo_slices_compact(%s)", args[0])
				},
				RuntimeHelpers: slicesHelpers,
			},
		},
	})
}
