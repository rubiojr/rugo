package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "crypto/sha256",
		Doc:  "SHA-256 hashing function from Go's crypto/sha256 package.",
		Funcs: map[string]GoFuncSig{
			"sum256": {
				GoName: "Sum256", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Returns the hex-encoded SHA-256 hash of a string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _h := %s.Sum256([]byte(%s)); return interface{}(fmt.Sprintf(\"%%x\", _h)) }()",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
		},
	})

	Register(&Package{
		Path: "crypto/md5",
		Doc:  "MD5 hashing function from Go's crypto/md5 package.",
		Funcs: map[string]GoFuncSig{
			"sum": {
				GoName: "Sum", Params: []GoType{GoString}, Returns: []GoType{GoString},
				Doc: "Returns the hex-encoded MD5 hash of a string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _h := %s.Sum([]byte(%s)); return interface{}(fmt.Sprintf(\"%%x\", _h)) }()",
						pkgBase, TypeConvToGo(args[0], GoString))
				},
			},
		},
	})
}
