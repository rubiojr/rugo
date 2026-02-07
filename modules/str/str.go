package strmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "str",
		Type: "Str",
		Funcs: []modules.FuncDef{
			{Name: "contains", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "split", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "trim", Args: []modules.ArgType{modules.String}},
			{Name: "starts_with", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "ends_with", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "replace", Args: []modules.ArgType{modules.String, modules.String, modules.String}},
			{Name: "upper", Args: []modules.ArgType{modules.String}},
			{Name: "lower", Args: []modules.ArgType{modules.String}},
			{Name: "index", Args: []modules.ArgType{modules.String, modules.String}},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
