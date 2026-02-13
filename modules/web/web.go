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
		Doc:  "HTTP web server with routing, middleware, and response helpers.",
		Funcs: []modules.FuncDef{
			{Name: "get", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Register a GET route with a path and handler."},
			{Name: "post", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Register a POST route with a path and handler."},
			{Name: "put", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Register a PUT route with a path and handler."},
			{Name: "delete", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Register a DELETE route with a path and handler."},
			{Name: "patch", Args: []modules.ArgType{modules.String, modules.String}, Variadic: true, Doc: "Register a PATCH route with a path and handler."},
			{Name: "middleware", Args: []modules.ArgType{modules.String}, Doc: "Register a middleware handler by name."},
			{Name: "rate_limit", Args: []modules.ArgType{modules.Any}, Doc: "Set the maximum requests per second for rate limiting."},
			{Name: "group", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Start a route group with a path prefix."},
			{Name: "end_group", Args: nil, Doc: "End the current route group."},
			{Name: "static", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Serve static files from a directory at the given path."},
			{Name: "listen", Args: []modules.ArgType{modules.Int}, Doc: "Start the web server on the given port."},
			{Name: "port", Args: nil, Doc: "Return the port the server is listening on."},
			{Name: "free_port", Args: nil, Doc: "Find and return an available port number."},
			{Name: "text", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Send a plain text response."},
			{Name: "html", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Send an HTML response."},
			{Name: "json", Args: []modules.ArgType{modules.Any}, Variadic: true, Doc: "Send a JSON response."},
			{Name: "redirect", Args: []modules.ArgType{modules.String}, Variadic: true, Doc: "Send a redirect response to the given URL."},
			{Name: "status", Args: []modules.ArgType{modules.Int}, Doc: "Set the HTTP status code for the response."},
		},
		GoImports:         []string{"encoding/json", "fmt", "io", "log", "math", "net", "net/http", "os", "path/filepath", "strings", "sync", "time"},
		DispatchEntry:     "listen",
		DispatchTransform: func(s string) string { return s },
		Runtime:           modules.CleanRuntime(runtime),
	})
}
