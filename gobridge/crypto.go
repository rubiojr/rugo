package gobridge

func init() {
	Register(&Package{
		Path: "crypto/sha256",
		Doc:  "SHA-256 hashing function from Go's crypto/sha256 package.",
		Funcs: map[string]GoFuncSig{
			"sum256": {
				GoName: "Sum256", Params: []GoType{GoByteSlice}, Returns: []GoType{GoByteSlice},
				Doc:        "Returns the raw SHA-256 hash of a string.",
				ArrayTypes: map[int]*GoArrayType{0: {Elem: GoByte, Size: 32}},
			},
		},
	})

	Register(&Package{
		Path: "crypto/md5",
		Doc:  "MD5 hashing function from Go's crypto/md5 package.",
		Funcs: map[string]GoFuncSig{
			"sum": {
				GoName: "Sum", Params: []GoType{GoByteSlice}, Returns: []GoType{GoByteSlice},
				Doc:        "Returns the raw MD5 hash of a string.",
				ArrayTypes: map[int]*GoArrayType{0: {Elem: GoByte, Size: 16}},
			},
		},
	})
}
