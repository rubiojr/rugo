package main

import (
	"github.com/rubiojr/rugo/cmd"
	_ "github.com/rubiojr/rugo/modules/ast"
	_ "github.com/rubiojr/rugo/modules/bench"
	_ "github.com/rubiojr/rugo/modules/cli"
	_ "github.com/rubiojr/rugo/modules/color"
	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/fmt"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/json"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/queue"
	_ "github.com/rubiojr/rugo/modules/re"
	_ "github.com/rubiojr/rugo/modules/sqlite"
	_ "github.com/rubiojr/rugo/modules/str"
	_ "github.com/rubiojr/rugo/modules/test"
	_ "github.com/rubiojr/rugo/modules/web"
)

var version = "v0.14.4"

func main() {
	cmd.Execute(version)
}
