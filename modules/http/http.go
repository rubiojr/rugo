package httpmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "http",
		Type: "HTTP",
		Funcs: []modules.FuncDef{
			{Name: "get", Args: []modules.ArgType{modules.String}},
			{Name: "post", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
		},
		GoImports: []string{"io", "net/http"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
