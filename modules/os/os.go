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
		Doc:  "Operating system operations: commands, files, environment, and process control.",
		Funcs: []modules.FuncDef{
			{Name: "exec", Args: []modules.ArgType{modules.String}, Doc: "Execute a shell command and return its output."},
			{Name: "exit", Args: []modules.ArgType{modules.Int}, Doc: "Exit the program with the given status code."},
			{Name: "file_exists", Args: []modules.ArgType{modules.String}, Doc: "Return true if the file or directory exists."},
			{Name: "is_dir", Args: []modules.ArgType{modules.String}, Doc: "Return true if the path exists and is a directory."},
			{Name: "read_line", Args: []modules.ArgType{modules.String}, Doc: "Print the prompt and read a line from stdin. Returns the input without trailing newline."},
			{Name: "getenv", Args: []modules.ArgType{modules.String}, Doc: "Get the value of an environment variable."},
			{Name: "setenv", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Set an environment variable."},
			{Name: "cwd", Args: []modules.ArgType{}, Doc: "Return the current working directory."},
			{Name: "chdir", Args: []modules.ArgType{modules.String}, Doc: "Change the current working directory."},
			{Name: "hostname", Args: []modules.ArgType{}, Doc: "Return the machine hostname."},
			{Name: "read_file", Args: []modules.ArgType{modules.String}, Doc: "Read the entire contents of a file as a string."},
			{Name: "write_file", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"path", "content"}, Doc: "Write a string to a file, creating or overwriting it."},
			{Name: "remove", Args: []modules.ArgType{modules.String}, Doc: "Remove a file or directory (recursive)."},
			{Name: "mkdir", Args: []modules.ArgType{modules.String}, Doc: "Create a directory and any necessary parents."},
			{Name: "rename", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"old", "new"}, Doc: "Rename or move a file or directory."},
			{Name: "glob", Args: []modules.ArgType{modules.String}, Doc: "Return an array of file paths matching a glob pattern."},
			{Name: "tmp_dir", Args: []modules.ArgType{}, Doc: "Return the default temporary directory path."},
			{Name: "args", Args: []modules.ArgType{}, Doc: "Return command-line arguments as an array."},
			{Name: "pid", Args: []modules.ArgType{}, Doc: "Return the current process ID."},
			{Name: "symlink", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"target", "link"}, Doc: "Create a symbolic link."},
			{Name: "readlink", Args: []modules.ArgType{modules.String}, Doc: "Return the target of a symbolic link."},
		},
		GoImports: []string{"bufio", "path/filepath"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
