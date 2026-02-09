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
		Doc:  "Regular expression matching, searching, and replacing.",
		Funcs: []modules.FuncDef{
			{Name: "test", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return true if the pattern matches the string."},
			{Name: "find", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return the first match of the pattern in the string."},
			{Name: "find_all", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return all matches of the pattern in the string."},
			{Name: "replace", Args: []modules.ArgType{modules.String, modules.String, modules.String}, Doc: "Replace the first match of the pattern with a replacement."},
			{Name: "replace_all", Args: []modules.ArgType{modules.String, modules.String, modules.String}, Doc: "Replace all matches of the pattern with a replacement."},
			{Name: "split", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Split a string by a regex pattern."},
			{Name: "match", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return captured groups from the first match."},
		},
		GoImports: []string{"regexp"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
