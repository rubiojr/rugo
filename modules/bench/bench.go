package benchmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name:    "bench",
		Type:    "Bench",
		Doc:     "Benchmarking framework for measuring code performance.",
		Funcs:   []modules.FuncDef{},
		Runtime: modules.CleanRuntime(runtime),
	})
}
