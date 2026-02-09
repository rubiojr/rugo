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
		Doc:  "HTTP client for making web requests.",
		Funcs: []modules.FuncDef{
			{Name: "get", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Send an HTTP GET request to a URL."},
			{Name: "post", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Send an HTTP POST request with a body."},
			{Name: "put", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Send an HTTP PUT request with a body."},
			{Name: "patch", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Send an HTTP PATCH request with a body."},
			{Name: "delete", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Send an HTTP DELETE request to a URL."},
		},
		GoImports: []string{"errors", "io", "net", "net/http", "net/url"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
