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
		Doc:  "Thread-safe queue for concurrent producer-consumer patterns.",
		Funcs: []modules.FuncDef{
			{Name: "new", Args: []modules.ArgType{}, Variadic: true, Doc: "Create a new thread-safe queue with optional capacity."},
		},
		GoImports: []string{"sync/atomic", "time"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
