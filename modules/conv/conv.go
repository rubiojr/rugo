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
			{Name: "to_bool", Args: []modules.ArgType{modules.Any}, Doc: "Convert a value to a boolean."},
			{Name: "parse_int", Args: []modules.ArgType{modules.String, modules.Int}, Doc: "Parse a string as an integer with a given base (e.g. 16 for hex)."},
		},
		GoImports: []string{"strconv"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
