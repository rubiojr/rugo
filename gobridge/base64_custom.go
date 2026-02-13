// Curated short names for base64 convenience API.
package gobridge

func init() {
	Extend("encoding/base64", map[string]GoFuncSig{
		"encode":     {GoName: "StdEncoding.EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to standard Base64."},
		"decode":     {GoName: "StdEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a standard Base64 string."},
		"url_encode": {GoName: "URLEncoding.EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to URL-safe Base64."},
		"url_decode": {GoName: "URLEncoding.DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a URL-safe Base64 string."},
	})
}
