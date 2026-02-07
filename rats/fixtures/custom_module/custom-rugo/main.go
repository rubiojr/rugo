package main

import (
	"github.com/rubiojr/rugo/cmd"

	// Standard Rugo modules
	_ "github.com/rubiojr/rugo/modules/cli"
	_ "github.com/rubiojr/rugo/modules/color"
	_ "github.com/rubiojr/rugo/modules/conv"
	_ "github.com/rubiojr/rugo/modules/http"
	_ "github.com/rubiojr/rugo/modules/json"
	_ "github.com/rubiojr/rugo/modules/os"
	_ "github.com/rubiojr/rugo/modules/str"
	_ "github.com/rubiojr/rugo/modules/test"

	// Custom module
	_ "github.com/rubiojr/rugo/rats/fixtures/custom_module/module"
)

func main() {
	cmd.Execute("v0.0.0-test")
}
