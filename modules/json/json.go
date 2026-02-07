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
		Funcs: []modules.FuncDef{
			{Name: "parse", Args: []modules.ArgType{modules.String}},
			{Name: "encode", Args: []modules.ArgType{modules.Any}},
		},
		GoImports: []string{"encoding/json", "math"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
