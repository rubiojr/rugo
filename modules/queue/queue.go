package queuemod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "queue",
		Type: "Queue",
		Funcs: []modules.FuncDef{
			{Name: "new", Args: []modules.ArgType{}, Variadic: true},
		},
		GoImports: []string{"sync/atomic", "time"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
