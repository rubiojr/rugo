package gobridge

func init() {
	Register(&Package{
		Path: "encoding/hex",
		Doc:  "Hexadecimal encoding and decoding functions from Go's encoding/hex package.",
		Funcs: map[string]GoFuncSig{
			"encode": {GoName: "EncodeToString", Params: []GoType{GoByteSlice}, Returns: []GoType{GoString}, Doc: "Encodes a string to hexadecimal."},
			"decode": {GoName: "DecodeString", Params: []GoType{GoString}, Returns: []GoType{GoByteSlice, GoError}, Doc: "Decodes a hexadecimal string."},
		},
	})
}
