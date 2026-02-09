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
		Doc:  "Testing framework with assertions for RATS test files.",
		GoImports: []string{
			"strconv",
			"time",
		},
		Funcs: []modules.FuncDef{
			{Name: "run", Args: []modules.ArgType{modules.String}, Doc: "Run a named test case."},
			{Name: "tmpdir", Args: []modules.ArgType{}, Doc: "Create and return a temporary directory path."},
			{Name: "write_file", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Write content to a file at the given path."},
			{Name: "assert_eq", Args: []modules.ArgType{modules.Any, modules.Any}, Doc: "Assert that two values are equal."},
			{Name: "assert_neq", Args: []modules.ArgType{modules.Any, modules.Any}, Doc: "Assert that two values are not equal."},
			{Name: "assert_true", Args: []modules.ArgType{modules.Any}, Doc: "Assert that a value is true."},
			{Name: "assert_false", Args: []modules.ArgType{modules.Any}, Doc: "Assert that a value is false."},
			{Name: "assert_contains", Args: []modules.ArgType{modules.Any, modules.Any}, Doc: "Assert that a string or array contains the given value."},
			{Name: "assert_nil", Args: []modules.ArgType{modules.Any}, Doc: "Assert that a value is nil."},
			{Name: "fail", Args: []modules.ArgType{modules.Any}, Doc: "Fail the test with a message."},
			{Name: "skip", Args: []modules.ArgType{modules.Any}, Doc: "Skip the current test with a reason."},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
