package astmod

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	dep, replace := rugoDepInfo()

	mod := &modules.Module{
		Name: "ast",
		Type: "AST",
		Doc:  "Parse and inspect Rugo source files.\n\n    prog = ast.parse_file(\"lib.rugo\")\n    for stmt in prog[\"statements\"]\n      if stmt[\"type\"] == \"def\"\n        puts(stmt[\"name\"])\n      end\n    end",
		Funcs: []modules.FuncDef{
			{Name: "parse_file", Args: []modules.ArgType{modules.String}, ArgNames: []string{"path"}, Doc: "Parse a .rugo file and return a program hash.\nKeys: \"source_file\", \"raw_source\", \"statements\", \"structs\"."},
			{Name: "parse_source", Args: []modules.ArgType{modules.String, modules.String}, ArgNames: []string{"source", "name"}, Doc: "Parse Rugo source code and return a program hash.\nThe name is used in error messages."},
			{Name: "source_lines", Args: []modules.ArgType{modules.Any, modules.Any}, ArgNames: []string{"program", "statement"}, Doc: "Extract the raw source lines for a statement.\nReturns an array of strings from the program's raw source."},
		},
		GoImports: []string{"github.com/rubiojr/rugo/compiler"},
		GoDeps:    []string{dep},
		Runtime:   modules.CleanRuntime(runtime),
	}
	if replace != "" {
		mod.GoReplace = []string{replace}
	}
	modules.Register(mod)
}

// rugoDepInfo returns the require line and optional replace directive for the
// rugo module. When running from a development build, it returns a replace
// directive pointing to the local source tree.
func rugoDepInfo() (dep string, replace string) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "github.com/rubiojr/rugo v0.0.0", ""
	}

	// When running from source (go run / go build in the module), the main
	// module is github.com/rubiojr/rugo itself.
	if bi.Main.Path == "github.com/rubiojr/rugo" {
		version := bi.Main.Version
		if version == "(devel)" || version == "" {
			version = "v0.0.0"
		}
		dep = "github.com/rubiojr/rugo " + version
		// Find the module root by walking up from the executable
		exe, err := os.Executable()
		if err == nil {
			dir := filepath.Dir(exe)
			// Check if go.mod exists in common locations relative to the binary
			for _, candidate := range []string{dir, filepath.Dir(dir)} {
				if _, err := os.Stat(filepath.Join(candidate, "go.mod")); err == nil {
					replace = "github.com/rubiojr/rugo => " + candidate
					return
				}
			}
		}
		return
	}

	// When installed as a dependency, use the recorded version.
	for _, m := range bi.Deps {
		if m.Path == "github.com/rubiojr/rugo" {
			dep = "github.com/rubiojr/rugo " + m.Version
			if m.Replace != nil {
				replace = "github.com/rubiojr/rugo => " + m.Replace.Path
			}
			return
		}
	}

	return "github.com/rubiojr/rugo v0.0.0", ""
}
