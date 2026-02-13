package gobridge

func init() {
	Register(&Package{
		Path: "encoding/base64",
		Doc:  "Base64 encoding and decoding functions from Go's encoding/base64 package.",
		Funcs: map[string]GoFuncSig{
			"encode":     {GoName: "StdEncoding.EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to standard Base64."},
			"decode":     {GoName: "StdEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a standard Base64 string."},
			"url_encode": {GoName: "URLEncoding.EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to URL-safe Base64."},
			"url_decode": {GoName: "URLEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a URL-safe Base64 string."},
		},
	})
}
