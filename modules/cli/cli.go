package climod

import (
	_ "embed"
	"strings"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "cli",
		Type: "CLI",
		Doc:  "Command-line argument parsing and subcommand routing.",
		Funcs: []modules.FuncDef{
			{Name: "name", Args: []modules.ArgType{modules.String}, Doc: "Set the application name."},
			{Name: "version", Args: []modules.ArgType{modules.String}, Doc: "Set the application version string."},
			{Name: "about", Args: []modules.ArgType{modules.String}, Doc: "Set the application description."},
			{Name: "cmd", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Register a subcommand with a name and description."},
			{Name: "flag", Args: []modules.ArgType{modules.String, modules.String, modules.String, modules.String, modules.String}, Doc: "Add a flag with command, name, short, description, and default value."},
			{Name: "bool_flag", Args: []modules.ArgType{modules.String, modules.String, modules.String, modules.String}, Doc: "Add a boolean flag with command, name, short, and description."},
			{Name: "run", Args: nil, Doc: "Parse arguments and dispatch to the matched subcommand handler."},
			{Name: "parse", Args: nil, Doc: "Parse arguments without dispatching."},
			{Name: "command", Args: nil, Doc: "Return the name of the matched subcommand."},
			{Name: "get", Args: []modules.ArgType{modules.String}, Doc: "Get the value of a parsed flag by name."},
			{Name: "args", Args: nil, Doc: "Return the remaining positional arguments after parsing."},
			{Name: "help", Args: nil, Doc: "Print the auto-generated help message."},
		},
		DispatchEntry:    "run",
		DispatchMainOnly: true,
		DispatchTransform: func(s string) string {
			return strings.NewReplacer(":", "_", "-", "_", " ", "_").Replace(s)
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
