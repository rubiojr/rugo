package climod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "cli",
		Type: "CLI",
		Funcs: []modules.FuncDef{
			{Name: "name", Args: []modules.ArgType{modules.String}},
			{Name: "version", Args: []modules.ArgType{modules.String}},
			{Name: "about", Args: []modules.ArgType{modules.String}},
			{Name: "cmd", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "flag", Args: []modules.ArgType{modules.String, modules.String, modules.String, modules.String, modules.String}},
			{Name: "bool_flag", Args: []modules.ArgType{modules.String, modules.String, modules.String, modules.String}},
			{Name: "run", Args: nil},
			{Name: "parse", Args: nil},
			{Name: "command", Args: nil},
			{Name: "get", Args: []modules.ArgType{modules.String}},
			{Name: "args", Args: nil},
			{Name: "help", Args: nil},
		},
		DispatchEntry: "run",
		Runtime:       modules.CleanRuntime(runtime),
	})
}
