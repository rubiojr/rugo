package evalmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	cacheDir, err := EnsureCompilerCache()
	if err != nil {
		modules.Register(&modules.Module{
			Name: "eval",
			Type: "Eval",
			Doc:  "Compile and run Rugo code at runtime.",
			Funcs: []modules.FuncDef{
				{Name: "run", Args: []modules.ArgType{modules.String}, ArgNames: []string{"source"}, Doc: "Compile and run Rugo source code. Returns a hash with status, output, and lines."},
				{Name: "file", Args: []modules.ArgType{modules.String}, Variadic: true, ArgNames: []string{"path"}, Doc: "Compile and run a Rugo source file. Returns a hash with status, output, and lines."},
			},
			Runtime: modules.CleanRuntime(runtime),
		})
		return
	}

	modules.Register(&modules.Module{
		Name: "eval",
		Type: "Eval",
		Doc:  "Compile and run Rugo code at runtime.\n\n    result = eval.run(\"puts 1 + 1\")\n    puts(result[\"output\"])\n    # => 2",
		Funcs: []modules.FuncDef{
			{Name: "run", Args: []modules.ArgType{modules.String}, ArgNames: []string{"source"}, Doc: "Compile and run Rugo source code.\nReturns a hash with keys: \"status\" (exit code), \"output\" (combined stdout/stderr), \"lines\" (array of output lines)."},
			{Name: "file", Args: []modules.ArgType{modules.String}, Variadic: true, ArgNames: []string{"path"}, Doc: "Compile and run a Rugo source file. Optional extra args are passed to the program.\nReturns a hash with keys: \"status\", \"output\", \"lines\"."},
		},
		GoImports: []string{"github.com/rubiojr/rugo/compiler", "path/filepath"},
		GoDeps:    []string{"github.com/rubiojr/rugo v0.0.0"},
		GoReplace: []string{"github.com/rubiojr/rugo => " + cacheDir},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
