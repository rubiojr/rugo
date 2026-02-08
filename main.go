package main

import (
	"github.com/rubiojr/rugo/cmd"
	_ "github.com/rubiojr/rugo/modules/cli"
	_ "github.com/rubiojr/rugo/modules/color"
	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/json"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/str"
	_ "github.com/rubiojr/rugo/modules/test"
	_ "github.com/rubiojr/rugo/modules/bench"
)

var version = "v0.4.0"

func main() {
	cmd.Execute(version)
}
