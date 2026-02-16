package base64mod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "base64",
		Type: "Base64",
		Doc:  "Base64 encoding and decoding.",
		Funcs: []modules.FuncDef{
			{Name: "encode", Args: []modules.ArgType{modules.String}, Doc: "Encode a string to standard base64."},
			{Name: "decode", Args: []modules.ArgType{modules.String}, Doc: "Decode a standard base64 string."},
			{Name: "url_encode", Args: []modules.ArgType{modules.String}, Doc: "Encode a string to URL-safe base64."},
			{Name: "url_decode", Args: []modules.ArgType{modules.String}, Doc: "Decode a URL-safe base64 string."},
		},
		GoImports: []string{"encoding/base64"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
