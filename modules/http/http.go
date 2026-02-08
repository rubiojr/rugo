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
			{Name: "get", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "post", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "put", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "patch", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "delete", Args: []modules.ArgType{modules.String}, Variadic: true},
		},
		GoImports: []string{"errors", "io", "net", "net/http", "net/url"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
