package osmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "os",
		Type: "OS",
		Funcs: []modules.FuncDef{
			{Name: "exec", Args: []modules.ArgType{modules.String}},
			{Name: "exit", Args: []modules.ArgType{modules.Int}},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
