package signalmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "signal",
		Type: "Signal",
		Doc:  "Race-free OS signal handling. Handlers run on the goroutine that calls signal.wait().",
		Funcs: []modules.FuncDef{
			{Name: "on", Args: []modules.ArgType{modules.String, modules.Any}, ArgNames: []string{"name", "handler"}, Doc: "Register a handler for a named signal (e.g. \"INT\", \"TERM\", \"HUP\"). The handler runs on the goroutine that calls signal.wait()."},
			{Name: "wait", Args: []modules.ArgType{}, Doc: "Block the calling goroutine, dispatching registered handlers as signals arrive. Handlers run on this goroutine, so they never race the main flow. Does not return; a handler must call exit() to terminate."},
			{Name: "reset", Args: []modules.ArgType{modules.String}, ArgNames: []string{"name"}, Doc: "Stop invoking the handler for a signal (undoes signal.on)."},
			{Name: "ignore", Args: []modules.ArgType{modules.String}, ArgNames: []string{"name"}, Doc: "Ignore a signal so it no longer affects the process."},
		},
		GoImports: []string{"os/signal", "sync", "syscall"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
