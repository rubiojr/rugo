// Curated aliases for rand convenience API.
package gobridge

func init() {
	Extend("math/rand/v2", map[string]GoFuncSig{
		"n": {GoName: "IntN", Params: []GoType{GoInt}, Returns: []GoType{GoInt}, Doc: "Alias for int_n. Returns a random int in [0, n)."},
	})
}
