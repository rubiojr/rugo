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
		Funcs: []modules.FuncDef{
			{Name: "sprintf", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "printf", Args: []modules.ArgType{modules.String}, Variadic: true},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
