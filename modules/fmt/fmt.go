package fmtmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "fmt",
		Type: "Fmt",
		Doc:  "Formatted string output using printf-style verbs.",
		Funcs: []modules.FuncDef{
			{Name: "sprintf", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Return a formatted string without printing it."},
			{Name: "printf", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Print a formatted string to stdout."},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
