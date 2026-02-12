package strmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "str",
		Type: "Str",
		Doc:  "String manipulation and searching.",
		Funcs: []modules.FuncDef{
			{Name: "contains", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return true if the string contains the substring."},
			{Name: "split", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Split a string by a separator into an array."},
			{Name: "trim", Args: []modules.ArgType{modules.String}, Doc: "Remove leading and trailing whitespace."},
			{Name: "starts_with", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return true if the string starts with the prefix."},
			{Name: "ends_with", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return true if the string ends with the suffix."},
			{Name: "replace", Args: []modules.ArgType{modules.String, modules.String, modules.String}, Doc: "Replace all occurrences of old with new in the string."},
			{Name: "upper", Args: []modules.ArgType{modules.String}, Doc: "Convert a string to uppercase."},
			{Name: "lower", Args: []modules.ArgType{modules.String}, Doc: "Convert a string to lowercase."},
			{Name: "index", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return the index of the first occurrence of the substring, or -1."},
			{Name: "join", Args: []modules.ArgType{modules.Any, modules.String}, Doc: "Join an array of strings with a separator."},
			{Name: "rune_count", Args: []modules.ArgType{modules.String}, Doc: "Return the number of Unicode characters (runes) in a string."},
		},
		GoImports: []string{"unicode/utf8"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
