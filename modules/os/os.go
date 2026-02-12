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
			{Name: "file_exists", Args: []modules.ArgType{modules.String}, Doc: "Return true if the file or directory exists."},
			{Name: "is_dir", Args: []modules.ArgType{modules.String}, Doc: "Return true if the path exists and is a directory."},
			{Name: "read_line", Args: []modules.ArgType{modules.String}, Doc: "Print the prompt and read a line from stdin. Returns the input without trailing newline."},
		},
		GoImports: []string{"bufio"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
