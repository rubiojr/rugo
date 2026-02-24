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
			{Name: "count", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Count non-overlapping occurrences of a substring."},
			{Name: "repeat", Args: []modules.ArgType{modules.String, modules.Int}, Doc: "Repeat a string n times."},
			{Name: "reverse", Args: []modules.ArgType{modules.String}, Doc: "Reverse a string by Unicode characters."},
			{Name: "chars", Args: []modules.ArgType{modules.String}, Doc: "Split a string into an array of individual characters."},
			{Name: "fields", Args: []modules.ArgType{modules.String}, Doc: "Split a string by whitespace into an array of words."},
			{Name: "trim_prefix", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Remove a prefix from a string if present."},
			{Name: "trim_suffix", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Remove a suffix from a string if present."},
			{Name: "pad_left", Args: []modules.ArgType{modules.String, modules.Int}, Variadic: true, Doc: "Left-pad a string to a given width. Optional third arg is the pad character."},
			{Name: "pad_right", Args: []modules.ArgType{modules.String, modules.Int}, Variadic: true, Doc: "Right-pad a string to a given width. Optional third arg is the pad character."},
			{Name: "each_line", Args: []modules.ArgType{modules.String}, Doc: "Split a string into an array of lines."},
			{Name: "center", Args: []modules.ArgType{modules.String, modules.Int}, Variadic: true, Doc: "Center a string within a given width. Optional third arg is the pad character."},
			{Name: "last_index", Args: []modules.ArgType{modules.String, modules.String}, Doc: "Return the index of the last occurrence of the substring, or -1."},
			{Name: "slice", Args: []modules.ArgType{modules.String, modules.Int, modules.Int}, Doc: "Extract a substring by rune start and end indices. Supports negative indices."},
			{Name: "empty", Args: []modules.ArgType{modules.String}, Doc: "Return true if the string is empty."},
			{Name: "byte_size", Args: []modules.ArgType{modules.String}, Doc: "Return the byte length of a string (not character count)."},
		},
		GoImports: []string{"unicode/utf8"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
