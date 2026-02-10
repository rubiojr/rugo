# Custom Modules (Advanced)

Rugo supports **custom modules** — you can create your own Rugo modules in Go
and build a custom Rugo binary that includes them, without modifying the Rugo
codebase.

This uses the same pattern as tools like Caddy and Hugo: your custom binary
imports the Rugo CLI package plus your module packages.

## Creating a Custom Module

A Rugo module is a Go package that calls `modules.Register()` in its `init()`
function. You need two files:

**runtime.go** — the Go implementation:

```go
//go:build ignore

package hello

type Hello struct{}

func (*Hello) Greet(name string) interface{} {
    return "hello, " + name
}
```

**hello.go** — the module registration:

```go
package hello

import (
    _ "embed"
    "github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
    modules.Register(&modules.Module{
        Name: "hello",
        Type: "Hello",
        Funcs: []modules.FuncDef{
            {Name: "greet", Args: []modules.ArgType{modules.String}},
        },
        Runtime: modules.CleanRuntime(runtime),
    })
}
```

Methods use typed parameters and PascalCase naming (`greet` → `Greet`).

## Building a Custom Rugo

Create a `main.go` that imports your module alongside the standard ones:

```go
package main

import (
    "github.com/rubiojr/rugo/cmd"

    // Standard modules
    _ "github.com/rubiojr/rugo/modules/cli"
    _ "github.com/rubiojr/rugo/modules/color"
    _ "github.com/rubiojr/rugo/modules/conv"
    _ "github.com/rubiojr/rugo/modules/http"
    _ "github.com/rubiojr/rugo/modules/json"
    _ "github.com/rubiojr/rugo/modules/os"
    _ "github.com/rubiojr/rugo/modules/str"
    _ "github.com/rubiojr/rugo/modules/test"

    // Your custom module
    _ "github.com/yourorg/rugo-hello"
)

func main() { cmd.Execute("v1.0.0-custom") }
```

Then build and use it:

```bash
go build -o myrugo .
./myrugo script.rugo
```

## Using Custom Modules in Scripts

```ruby
use "hello"

puts hello.greet("developer")   # hello, developer
```

Your custom Rugo binary works exactly like standard Rugo — it just has
extra modules available. Note: custom modules use `use` just like the
built-in stdlib modules.

## Wrapping External Go Libraries

Modules can wrap any Go package. Declare `GoDeps` so the generated programs
can resolve external dependencies:

```go
modules.Register(&modules.Module{
    Name: "slug",
    Type: "Slug",
    Funcs: []modules.FuncDef{
        {Name: "make", Args: []modules.ArgType{modules.String}},
    },
    GoImports: []string{`gosimpleslug "github.com/gosimple/slug"`},
    GoDeps:    []string{"github.com/gosimple/slug v1.15.0"},
    Runtime:   modules.CleanRuntime(runtime),
})
```

See `examples/modules/slug/` for a complete working example.

---
For the full module API reference, see the [Modules Reference](../mods.md).
