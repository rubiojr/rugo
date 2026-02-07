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
)

var version = "v0.2.5"

func main() {
	cmd.Execute(version)
}
