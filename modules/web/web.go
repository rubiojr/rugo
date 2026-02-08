package webmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "web",
		Type: "Web",
		Funcs: []modules.FuncDef{
			{Name: "get", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "post", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "put", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "delete", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "patch", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true},
			{Name: "middleware", Args: []modules.ArgType{modules.String}},
			{Name: "rate_limit", Args: []modules.ArgType{modules.Any}},
			{Name: "group", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "end_group", Args: nil},
			{Name: "static", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "listen", Args: []modules.ArgType{modules.Int}},
			{Name: "text", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "html", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "json", Args: []modules.ArgType{modules.Any}, Variadic: true},
			{Name: "redirect", Args: []modules.ArgType{modules.String}, Variadic: true},
			{Name: "status", Args: []modules.ArgType{modules.Int}},
		},
		GoImports:     []string{"encoding/json", "fmt", "io", "log", "math", "net", "net/http", "os", "path/filepath", "strings", "sync", "time"},
		DispatchEntry: "listen",
		Runtime:       modules.CleanRuntime(runtime),
	})
}
