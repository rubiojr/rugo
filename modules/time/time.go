package timemod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "time",
		Type: "Time",
		Doc:  "Time operations: timestamps, sleeping, formatting, and parsing.",
		Funcs: []modules.FuncDef{
			{Name: "now", Args: []modules.ArgType{}, Doc: "Return the current Unix timestamp as a float with nanosecond precision."},
			{Name: "sleep", Args: []modules.ArgType{modules.Float}, Doc: "Sleep for the given number of seconds (float)."},
			{Name: "format", Args: []modules.ArgType{modules.Float, modules.String}, ArgNames: []string{"timestamp", "layout"}, Doc: "Format a Unix timestamp using a Go time layout string."},
			{Name: "parse", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"str", "layout"}, Doc: "Parse a time string using a Go time layout, returning a Unix timestamp float."},
			{Name: "since", Args: []modules.ArgType{modules.Float}, Doc: "Return seconds elapsed since the given Unix timestamp."},
			{Name: "millis", Args: []modules.ArgType{}, Doc: "Return the current time in milliseconds as an integer."},
		},
		GoImports: []string{"time"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
