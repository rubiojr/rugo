package convmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "conv",
		Type: "Conv",
		Doc:  "Type conversion between strings, integers, and floats.",
		Funcs: []modules.FuncDef{
			{Name: "to_i", Args: []modules.ArgType{modules.Any}, Doc: "Convert a value to an integer."},
			{Name: "to_f", Args: []modules.ArgType{modules.Any}, Doc: "Convert a value to a float."},
			{Name: "to_s", Args: []modules.ArgType{modules.Any}, Doc: "Convert a value to a string."},
		},
		GoImports: []string{"strconv"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
