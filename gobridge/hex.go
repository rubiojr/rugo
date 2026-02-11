package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "encoding/hex",
		Doc:  "Hexadecimal encoding and decoding functions from Go's encoding/hex package.",
		Funcs: map[string]GoFuncSig{
			"encode": {
				GoName: "EncodeToString", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Encodes a string to hexadecimal.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("interface{}(%s.EncodeToString([]byte(%s)))",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
			"decode": {
				GoName: "DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Decodes a hexadecimal string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.DecodeString(%s); if _err != nil { %s }; return interface{}(string(_v)) }()",
						pkgBase, TypeConvToGo(args[0], GoString), PanicOnErr(rugoName))
				},
			},
		},
	})
}
