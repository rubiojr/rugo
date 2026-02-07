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
		Funcs: []modules.FuncDef{
			{Name: "to_i", Args: []modules.ArgType{modules.Any}},
			{Name: "to_f", Args: []modules.ArgType{modules.Any}},
			{Name: "to_s", Args: []modules.ArgType{modules.Any}},
		},
		GoImports: []string{"strconv"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
