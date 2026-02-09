package jsonmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "json",
		Type: "JSON",
		Doc:  "JSON encoding and decoding.",
		Funcs: []modules.FuncDef{
			{Name: "parse", Args: []modules.ArgType{modules.String}, Doc: "Parse a JSON string into a hash or array."},
			{Name: "encode", Args: []modules.ArgType{modules.Any}, Doc: "Encode a value as a JSON string."},
		},
		GoImports: []string{"encoding/json", "math"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
