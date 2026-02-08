package remod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "re",
		Type: "Re",
		Funcs: []modules.FuncDef{
			{Name: "test", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "find", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "find_all", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "replace", Args: []modules.ArgType{modules.String, modules.String, modules.String}},
			{Name: "replace_all", Args: []modules.ArgType{modules.String, modules.String, modules.String}},
			{Name: "split", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "match", Args: []modules.ArgType{modules.String, modules.String}},
		},
		GoImports: []string{"regexp"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
