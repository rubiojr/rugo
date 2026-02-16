package filepathmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "filepath",
		Type: "Filepath",
		Doc:  "File path manipulation and querying.",
		Funcs: []modules.FuncDef{
			{Name: "join", Args: []modules.ArgType{}, Variadic: true, Doc: "Join path segments into a single path."},
			{Name: "base", Args: []modules.ArgType{modules.String}, Doc: "Return the last element of a path."},
			{Name: "dir", Args: []modules.ArgType{modules.String}, Doc: "Return all but the last element of a path."},
			{Name: "ext", Args: []modules.ArgType{modules.String}, Doc: "Return the file extension."},
			{Name: "abs", Args: []modules.ArgType{modules.String}, Doc: "Return the absolute path."},
			{Name: "rel", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return a relative path from base to target."},
			{Name: "glob", Args: []modules.ArgType{modules.String}, Doc: "Return files matching a glob pattern."},
			{Name: "clean", Args: []modules.ArgType{modules.String}, Doc: "Return the shortest equivalent path."},
			{Name: "is_abs", Args: []modules.ArgType{modules.String}, Doc: "Return true if the path is absolute."},
			{Name: "split", Args: []modules.ArgType{modules.String}, Doc: "Split a path into directory and file components."},
			{Name: "match", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return true if the name matches the glob pattern."},
		},
		GoImports: []string{"path/filepath"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
