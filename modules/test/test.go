package testmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "test",
		Type: "Test",
		GoImports: []string{
			"strconv",
			"time",
		},
		Funcs: []modules.FuncDef{
			{Name: "run", Args: []modules.ArgType{modules.String}},
			{Name: "tmpdir", Args: []modules.ArgType{}},
			{Name: "write_file", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "assert_eq", Args: []modules.ArgType{modules.Any, modules.Any}},
			{Name: "assert_neq", Args: []modules.ArgType{modules.Any, modules.Any}},
			{Name: "assert_true", Args: []modules.ArgType{modules.Any}},
			{Name: "assert_false", Args: []modules.ArgType{modules.Any}},
			{Name: "assert_contains", Args: []modules.ArgType{modules.Any, modules.Any}},
			{Name: "assert_nil", Args: []modules.ArgType{modules.Any}},
			{Name: "fail", Args: []modules.ArgType{modules.Any}},
			{Name: "skip", Args: []modules.ArgType{modules.Any}},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
