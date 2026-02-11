package astmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/ast"
	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	cacheDir, err := ast.EnsureCache()
	if err != nil {
		// Cache failure is not fatal at init time; it will fail at build time
		// with a clearer error. Register the module without deps.
		modules.Register(&modules.Module{
			Name: "ast",
			Type: "AST",
			Doc:  "Parse and inspect Rugo source files.",
			Funcs: []modules.FuncDef{
				{Name: "parse_file", Args: []modules.ArgType{modules.String}, ArgNames: []string{"path"}, Doc: "Parse a .rugo file and return a program hash."},
				{Name: "parse_source", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"source", "name"}, Doc: "Parse Rugo source code and return a program hash."},
				{Name: "source_lines", Args: []modules.ArgType{modules.Any, modules.Any}, ArgNames: []string{"program", "statement"}, Doc: "Extract the raw source lines for a statement."},
			},
			Runtime: modules.CleanRuntime(runtime),
		})
		return
	}

	modules.Register(&modules.Module{
		Name: "ast",
		Type: "AST",
		Doc:  "Parse and inspect Rugo source files.\n\n    prog = ast.parse_file(\"lib.rugo\")\n    for stmt in prog[\"statements\"]\n      if stmt[\"type\"] == \"def\"\n        puts(stmt[\"name\"])\n      end\n    end",
		Funcs: []modules.FuncDef{
			{Name: "parse_file", Args: []modules.ArgType{modules.String}, ArgNames: []string{"path"}, Doc: "Parse a .rugo file and return a program hash.\nKeys: \"source_file\", \"raw_source\", \"statements\", \"structs\"."},
			{Name: "parse_source", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"source", "name"}, Doc: "Parse Rugo source code and return a program hash.\nThe name is used in error messages."},
			{Name: "source_lines", Args: []modules.ArgType{modules.Any, modules.Any}, ArgNames: []string{"program", "statement"}, Doc: "Extract the raw source lines for a statement.\nReturns an array of strings from the program's raw source."},
		},
		GoImports: []string{"github.com/rubiojr/rugo/ast"},
		GoDeps:    []string{"github.com/rubiojr/rugo v0.0.0"},
		GoReplace: []string{"github.com/rubiojr/rugo => " + cacheDir},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
