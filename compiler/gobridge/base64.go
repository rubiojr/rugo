package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "encoding/base64",
		Doc:  "Base64 encoding and decoding functions from Go's encoding/base64 package.",
		Funcs: map[string]GoFuncSig{
			"encode": {
				GoName: "StdEncoding.EncodeToString", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Encodes a string to standard Base64.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("interface{}(%s.StdEncoding.EncodeToString([]byte(%s)))",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
			"decode": {
				GoName: "StdEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Decodes a standard Base64 string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.StdEncoding.DecodeString(%s); if _err != nil { %s }; return interface{}(string(_v)) }()",
						pkgBase, TypeConvToGo(args[0], GoString), PanicOnErr(rugoName))
				},
			},
			"url_encode": {
				GoName: "URLEncoding.EncodeToString", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Encodes a string to URL-safe Base64.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("interface{}(%s.URLEncoding.EncodeToString([]byte(%s)))",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
			"url_decode": {
				GoName: "URLEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Decodes a URL-safe Base64 string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.URLEncoding.DecodeString(%s); if _err != nil { %s }; return interface{}(string(_v)) }()",
						pkgBase, TypeConvToGo(args[0], GoString), PanicOnErr(rugoName))
				},
			},
		},
	})
}
