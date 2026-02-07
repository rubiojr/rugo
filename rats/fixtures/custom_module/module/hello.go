package hello

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "hello",
		Type: "Hello",
		Funcs: []modules.FuncDef{
			{Name: "greet", Args: []modules.ArgType{modules.String}},
			{Name: "world"},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
