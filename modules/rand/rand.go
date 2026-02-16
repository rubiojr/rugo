package randmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "rand",
		Type: "Rand",
		Doc:  "Random number and string generation.",
		Funcs: []modules.FuncDef{
			{Name: "int", Args: []modules.ArgType{modules.Int, modules.Int}, ArgNames: []string{"min", "max"}, Doc: "Return a random integer in [min, max)."},
			{Name: "float", Args: []modules.ArgType{}, Doc: "Return a random float in [0.0, 1.0)."},
			{Name: "string", Args: []modules.ArgType{modules.Int}, ArgNames: []string{"length"}, Doc: "Return a random alphanumeric string of the given length."},
			{Name: "choice", Args: []modules.ArgType{modules.Any}, Doc: "Return a random element from an array."},
			{Name: "shuffle", Args: []modules.ArgType{modules.Any}, Doc: "Return a shuffled copy of an array."},
			{Name: "uuid", Args: []modules.ArgType{}, Doc: "Generate a random UUID v4 string."},
		},
		GoImports: []string{"math/rand/v2"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
