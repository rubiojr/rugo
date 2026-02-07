package slug

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "slug",
		Type: "Slug",
		Funcs: []modules.FuncDef{
			{Name: "make", Args: []modules.ArgType{modules.String}},
			{Name: "make_lang", Args: []modules.ArgType{modules.String, modules.String}},
			{Name: "is_slug", Args: []modules.ArgType{modules.String}},
			{Name: "join", Variadic: true},
		},
		GoImports: []string{
			`gosimpleslug "github.com/gosimple/slug"`,
		},
		GoDeps: []string{
			"github.com/gosimple/slug v1.15.0",
		},
		Runtime: modules.CleanRuntime(runtime),
	})
}
