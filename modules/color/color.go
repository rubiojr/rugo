package colormod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	s := modules.String
	modules.Register(&modules.Module{
		Name: "color",
		Type: "Color",
		Doc:  "ANSI color and style formatting for terminal output.",
		Funcs: []modules.FuncDef{
			// Foreground
			{Name: "red", Args: []modules.ArgType{s}, Doc: "Wrap text in red foreground color."},
			{Name: "green", Args: []modules.ArgType{s}, Doc: "Wrap text in green foreground color."},
			{Name: "yellow", Args: []modules.ArgType{s}, Doc: "Wrap text in yellow foreground color."},
			{Name: "blue", Args: []modules.ArgType{s}, Doc: "Wrap text in blue foreground color."},
			{Name: "magenta", Args: []modules.ArgType{s}, Doc: "Wrap text in magenta foreground color."},
			{Name: "cyan", Args: []modules.ArgType{s}, Doc: "Wrap text in cyan foreground color."},
			{Name: "white", Args: []modules.ArgType{s}, Doc: "Wrap text in white foreground color."},
			{Name: "gray", Args: []modules.ArgType{s}, Doc: "Wrap text in gray foreground color."},
			// Background
			{Name: "bg_red", Args: []modules.ArgType{s}, Doc: "Wrap text in red background color."},
			{Name: "bg_green", Args: []modules.ArgType{s}, Doc: "Wrap text in green background color."},
			{Name: "bg_yellow", Args: []modules.ArgType{s}, Doc: "Wrap text in yellow background color."},
			{Name: "bg_blue", Args: []modules.ArgType{s}, Doc: "Wrap text in blue background color."},
			{Name: "bg_magenta", Args: []modules.ArgType{s}, Doc: "Wrap text in magenta background color."},
			{Name: "bg_cyan", Args: []modules.ArgType{s}, Doc: "Wrap text in cyan background color."},
			{Name: "bg_white", Args: []modules.ArgType{s}, Doc: "Wrap text in white background color."},
			{Name: "bg_gray", Args: []modules.ArgType{s}, Doc: "Wrap text in gray background color."},
			// Styles
			{Name: "bold", Args: []modules.ArgType{s}, Doc: "Wrap text in bold style."},
			{Name: "dim", Args: []modules.ArgType{s}, Doc: "Wrap text in dim style."},
			{Name: "underline", Args: []modules.ArgType{s}, Doc: "Wrap text in underline style."},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
