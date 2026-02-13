// Curated short names for hex convenience API.
package gobridge

func init() {
	Extend("encoding/hex", map[string]GoFuncSig{
		"encode": {GoName: "EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to hexadecimal."},
		"decode": {GoName: "DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a hexadecimal string."},
	})
}
