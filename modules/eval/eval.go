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
		GoImports: []string{
			"github.com/rubiojr/rugo/compiler", "path/filepath",
			// Blank-import all stdlib modules so their init() functions
			// populate the module registry inside the compiled eval binary.
			// Excluded: ast (requires its own cache), eval (self-referential),
			// sqlite (requires external modernc.org/sqlite dependency).
			`_ "github.com/rubiojr/rugo/modules/base64"`,
			`_ "github.com/rubiojr/rugo/modules/bench"`,
			`_ "github.com/rubiojr/rugo/modules/cli"`,
			`_ "github.com/rubiojr/rugo/modules/color"`,
			`_ "github.com/rubiojr/rugo/modules/conv"`,
			`_ "github.com/rubiojr/rugo/modules/crypto"`,
			`_ "github.com/rubiojr/rugo/modules/filepath"`,
			`_ "github.com/rubiojr/rugo/modules/fmt"`,
			`_ "github.com/rubiojr/rugo/modules/hex"`,
			`_ "github.com/rubiojr/rugo/modules/http"`,
			`_ "github.com/rubiojr/rugo/modules/json"`,
			`_ "github.com/rubiojr/rugo/modules/math"`,
			`_ "github.com/rubiojr/rugo/modules/os"`,
			`_ "github.com/rubiojr/rugo/modules/queue"`,
			`_ "github.com/rubiojr/rugo/modules/rand"`,
			`_ "github.com/rubiojr/rugo/modules/re"`,
			`_ "github.com/rubiojr/rugo/modules/str"`,
			`_ "github.com/rubiojr/rugo/modules/test"`,
			`_ "github.com/rubiojr/rugo/modules/time"`,
			`_ "github.com/rubiojr/rugo/modules/web"`,
		},
		GoDeps:    []string{"github.com/rubiojr/rugo v0.0.0"},
		GoReplace: []string{"github.com/rubiojr/rugo => " + cacheDir},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
