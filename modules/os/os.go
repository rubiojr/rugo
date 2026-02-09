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
		Doc:  "Operating system operations like running commands and exiting.",
		Funcs: []modules.FuncDef{
			{Name: "exec", Args: []modules.ArgType{modules.String}, Doc: "Execute a shell command and return its output."},
			{Name: "exit", Args: []modules.ArgType{modules.Int}, Doc: "Exit the program with the given status code."},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
