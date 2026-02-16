package hexmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "hex",
		Type: "Hex",
		Doc:  "Hexadecimal encoding and decoding.",
		Funcs: []modules.FuncDef{
			{Name: "encode", Args: []modules.ArgType{modules.String}, Doc: "Encode a string to hexadecimal."},
			{Name: "decode", Args: []modules.ArgType{modules.String}, Doc: "Decode a hexadecimal string."},
		},
		GoImports: []string{"encoding/hex"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
