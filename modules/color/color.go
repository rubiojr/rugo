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
		Funcs: []modules.FuncDef{
			// Foreground
			{Name: "red", Args: []modules.ArgType{s}},
			{Name: "green", Args: []modules.ArgType{s}},
			{Name: "yellow", Args: []modules.ArgType{s}},
			{Name: "blue", Args: []modules.ArgType{s}},
			{Name: "magenta", Args: []modules.ArgType{s}},
			{Name: "cyan", Args: []modules.ArgType{s}},
			{Name: "white", Args: []modules.ArgType{s}},
			{Name: "gray", Args: []modules.ArgType{s}},
			// Background
			{Name: "bg_red", Args: []modules.ArgType{s}},
			{Name: "bg_green", Args: []modules.ArgType{s}},
			{Name: "bg_yellow", Args: []modules.ArgType{s}},
			{Name: "bg_blue", Args: []modules.ArgType{s}},
			{Name: "bg_magenta", Args: []modules.ArgType{s}},
			{Name: "bg_cyan", Args: []modules.ArgType{s}},
			{Name: "bg_white", Args: []modules.ArgType{s}},
			{Name: "bg_gray", Args: []modules.ArgType{s}},
			// Styles
			{Name: "bold", Args: []modules.ArgType{s}},
			{Name: "dim", Args: []modules.ArgType{s}},
			{Name: "underline", Args: []modules.ArgType{s}},
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
