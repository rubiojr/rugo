// Custom Rugo binary with the slug module.
//
// Build:
//
//	go build -o myrugo .
//	./myrugo ../example.rg
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
	_ "github.com/rubiojr/rugo/examples/modules/slug/module"
)

var version = "v0.1.5-custom"

func main() {
	cmd.Execute(version)
}
