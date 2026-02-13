# Rugo Module System

Rugo uses a registry-based module system. Each stdlib module lives in its own
directory under `modules/` and self-registers at startup via Go's `init()`.

## Built-in Modules

| Module | Functions | Description |
|--------|-----------|-------------|
| `ast`  | `parse_file`, `parse_source`, `source_lines` | Parse and inspect Rugo source files |
| `os`   | `exec`, `exit` | Shell execution and process control |
| `http` | `get`, `post`, `put`, `patch`, `delete` | HTTP client |
| `conv` | `to_i`, `to_f`, `to_s` | Type conversions |
| `cli`  | `name`, `version`, `about`, `cmd`, `flag`, `bool_flag`, `run`, `parse`, `command`, `get`, `args`, `help` | CLI app builder with commands, flags, and dispatch |
| `color` | `red`, `green`, `yellow`, `blue`, `magenta`, `cyan`, `white`, `gray`, `bg_*`, `bold`, `dim`, `underline` | ANSI terminal colors and styles |
| `json` | `parse`, `encode` | JSON parsing and encoding |
| `sqlite` | `open`, `exec`, `query`, `query_row`, `query_val`, `close` | SQLite database access |

### ast — Parse and Inspect Rugo Source

The `ast` module exposes the Rugo compiler API to Rugo scripts. It parses source files into hash-based ASTs for building linters, refactoring tools, and code analysis.

```ruby
use "ast"

prog = ast.parse_file("lib.rugo")
# Or parse a string:
# prog = ast.parse_source("def foo()\nend\n", "lib.rugo")

# Program hash keys: "source_file", "raw_source", "statements", "structs"
for stmt in prog["statements"]
  if stmt["type"] == "def"
    puts(stmt["name"] + " at line " + conv.to_s(stmt["line"]))
    lines = ast.source_lines(prog, stmt)
    puts("  " + conv.to_s(len(lines)) + " lines of source")
  end
end
```

**Functions:**

| Function | Description |
|----------|-------------|
| `ast.parse_file(path)` | Parse a `.rugo` file → program hash |
| `ast.parse_source(source, name)` | Parse a source string → program hash |
| `ast.source_lines(prog, stmt)` | Extract raw source lines for a statement |

**Statement types:** Each statement is a hash with `"type"`, `"line"`, `"end_line"`, and type-specific fields:

| Type | Fields |
|------|--------|
| `"def"` | `"name"`, `"params"` (array), `"body"` (array) |
| `"test"` | `"name"`, `"body"` |
| `"bench"` | `"name"`, `"body"` |
| `"if"` | `"body"`, `"elsif"`, `"else_body"` |
| `"while"` | `"body"` |
| `"for"` | `"var"`, `"index_var"` (optional), `"body"` |
| `"assign"` | `"target"` |
| `"return"` | — |
| `"break"` / `"next"` | — |
| `"expr"` | `"expr"` (expression hash) |
| `"use"` | `"module"` |
| `"require"` | `"path"`, `"alias"` (optional), `"with"` (optional) |
| `"import"` | `"package"`, `"alias"` (optional) |

### Usage in Rugo

```ruby
use "http"
use "conv"

result = `whoami`
puts result

resp = http.get("https://example.com")
puts resp.body
puts conv.to_s(42)
```

## Creating a New Module

### 1. Create the module directory

```
modules/mymod/
  mymod.go       # Registration
  runtime.go     # Runtime Go code (struct + methods)
  stubs.go       # Runtime helper stubs (for standalone compilation)
```

### 2. Write the runtime as typed Go code

`modules/mymod/runtime.go`:

```go
package mymod

import "fmt"

type MyMod struct{}

func (*MyMod) Hello(name string) interface{} {
    return "hello, " + name
}
```

Methods use **typed parameters** and are defined as pointer receiver methods on
a struct — no `interface{}` args, no module prefix. A package-level instance
(`var _mymod = &MyMod{}`) is generated automatically, so struct fields persist
across calls. The framework auto-generates wrappers that handle argument
validation and type conversion.

Method names are derived from rugo function names via snake_case → PascalCase
(`get` → `Get`, `to_s` → `ToS`, `my_func` → `MyFunc`).

Runtime files are normal Go code — they compile as part of the package and can
be tested directly. `CleanRuntime` strips the `package`, `import`, and
`//go:build` lines before embedding into generated programs.

Runtime functions can call base runtime helpers (`rugo_to_string`, `rugo_to_int`,
`rugo_to_float`, `rugo_to_bool`) — these are provided by `stubs.go` for
standalone compilation and by the Rugo runtime in generated programs.

### 3. Add runtime helper stubs (if needed)

If your runtime code calls `rugo_to_string`, `rugo_to_int`, `rugo_to_float`,
or `rugo_to_bool`, add a `stubs.go` that provides them so the module compiles
standalone and is testable:

`modules/mymod/stubs.go`:

```go
package mymod

import "fmt"

func rugo_to_string(v interface{}) string { return fmt.Sprintf("%v", v) }
```

Only include the helpers your runtime actually uses.

### 4. Register the module with typed function definitions

`modules/mymod/mymod.go`:

```go
package mymod

import (
    _ "embed"

    "github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
    modules.Register(&modules.Module{
        Name: "mymod",
        Type: "MyMod",
        Funcs: []modules.FuncDef{
            {Name: "hello", Args: []modules.ArgType{modules.String}},
        },
        // Additional Go imports beyond the base set (fmt, os, os/exec, strings).
        GoImports: []string{"encoding/json"},
        Runtime:   modules.CleanRuntime(runtime),
    })
}
```

The `Funcs` field describes each function's argument types. The `Type` field
names the Go struct. The framework generates a `var _mymod = &MyMod{}`
instance and a wrapper `rugo_mymod_hello` that validates args, converts
`args[0]` to `string`, and calls `_mymod.Hello(...)`.

### 5. Wire it up in `main.go`

Add a blank import so the module's `init()` runs:

```go
import (
    _ "github.com/rubiojr/rugo/modules/mymod"
)
```

### 6. Use it in Rugo

```ruby
use "mymod"

puts mymod.hello("developer")
```

## External Modules (Custom Rugo Builds)

You can create Rugo modules in **your own Go packages** — no changes to the
Rugo codebase needed. Then build a custom Rugo binary that includes your
modules alongside the standard ones.

This uses the same pattern as Caddy, Hugo, and other extensible Go tools.

### How it works

The Rugo CLI logic lives in the `cmd` package and is called via
`cmd.Execute(version)`. The default `main.go` simply imports the standard
modules and calls `Execute`. To add custom modules, create your own `main.go`
that imports your modules too.

### 1. Create a module package

Write a normal Rugo module in its own Go package (can be in any repo):

```go
// mymod/mymod.go
package mymod

import (
    _ "embed"
    "github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
    modules.Register(&modules.Module{
        Name: "mymod",
        Type: "MyMod",
        Funcs: []modules.FuncDef{
            {Name: "hello", Args: []modules.ArgType{modules.String}},
        },
        Runtime: modules.CleanRuntime(runtime),
    })
}
```

If your module depends on external Go packages, add `GoDeps` so the generated
programs can resolve them:

```go
modules.Register(&modules.Module{
    Name: "mymod",
    GoDeps: []string{"github.com/some/pkg v1.2.0"},
    GoImports: []string{`somepkg "github.com/some/pkg"`},
    // ...
})
```

### 2. Build a custom Rugo binary

Create a `main.go` that imports your module:

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
    _ "github.com/yourorg/rugo-mymod"
)

func main() { cmd.Execute("v1.0.0-custom") }
```

Then `go build -o myrugo .` and use it like normal Rugo.

### 3. Use it in Rugo scripts

```ruby
use "mymod"

puts mymod.hello("world")
```

See `examples/modules/slug/` for a complete working example that wraps
the [gosimple/slug](https://github.com/gosimple/slug) Go library.

## Go Modules via `require` (Lightweight)

For simpler cases that don't need state or the full module registration
system, you can write a standard Go package and `require` it directly.
The compiler introspects the Go source, classifies exported functions,
and bridges them automatically — no manifest, no `init()`, no custom binary.

### 1. Write a Go package

```go
// mymod/go.mod
module example.com/mymod

go 1.22
```

```go
// mymod/mymod.go
package mymod

func Greet(name string) string {
    return "hello, " + name
}

func Add(a int, b int) int {
    return a + b
}
```

### 2. Require it from Rugo

```ruby
require "mymod"

puts(mymod.greet("world"))   # hello, world
puts(mymod.add(3, 4))        # 7
```

### How it works

When `require` encounters a directory with `go.mod` and `.go` files (and no
`.rugo` entry point), it:

1. Parses the Go source with `go/types` (best-effort type checking)
2. Classifies exported functions using the same tier system as `bridgegen`
3. Registers them as a Go bridge package
4. Adds the module to the generated `go.mod` with a `replace` directive

### Supported types

Functions must use bridgeable parameter and return types:

| Go type    | Bridged as |
|------------|------------|
| `string`   | string     |
| `int`      | integer    |
| `float64`  | float      |
| `bool`     | boolean    |
| `error`    | auto-panic |
| `[]string` | array      |
| `[]byte`   | string     |

Functions with non-bridgeable types (pointers, interfaces, channels, maps,
structs, generics) are automatically excluded. If no functions are bridgeable,
the compiler reports a clear error listing each function and why it was blocked.

### Remote Go modules

This works with remote repositories too:

```ruby
require "github.com/user/rugo-slug@v1.0.0" as slug
slug.make("Hello World!")
```

### When to use `require` vs custom builds

| Feature | `require` (lightweight) | Custom build (full) |
|---------|------------------------|---------------------|
| Setup | None — just write Go | Build custom binary |
| State | Stateless (package funcs) | Stateful (struct methods) |
| Types | Basic types only | Any Go type |
| Dispatch | No | Yes (CLI/web handlers) |
| Dependencies | Automatic (from go.mod) | Via GoDeps field |

Use `require` for wrapping Go libraries. Use custom builds when you need
stateful modules, dispatch, or complex type handling.

## Module Struct Reference

```go
type ArgType int
const (
    String ArgType = iota
    Int
    Float
    Bool
    Any
)

type FuncDef struct {
    Name     string     // Rugo function name (e.g. "exec")
    Args     []ArgType  // Typed argument list
    Variadic bool       // Accept extra args beyond Args
}

type Module struct {
    Name          string      // Rugo import name (e.g. "os")
    Type          string      // Go struct type name (e.g. "OS", "HTTP")
    Funcs         []FuncDef   // Function definitions with typed args
    GoImports     []string    // Additional Go imports (e.g. ["net/http"])
    GoDeps        []string    // Go module deps (e.g. ["github.com/foo/bar v1.0.0"])
    Runtime       string      // Struct type + methods from runtime.go
    DispatchEntry string      // Module function that triggers dispatch (optional)
}
```

### Field details

**Name** — The string used in `use "name"` in Rugo source files.

**Funcs** — Describes each function exposed by the module. The `Args` field
declares typed parameters — the framework generates a wrapper that converts
`interface{}` arguments to the declared types and calls the corresponding
method on a persistent instance (e.g. `_os.Exec(...)`).

Available argument types:

| ArgType  | Go parameter type | Conversion |
|----------|-------------------|------------|
| `String` | `string`          | `rugo_to_string(args[i])` |
| `Int`    | `int`             | `rugo_to_int(args[i])` |
| `Float`  | `float64`         | `rugo_to_float(args[i])` |
| `Bool`   | `bool`            | `rugo_to_bool(args[i])` |
| `Any`    | `interface{}`     | `args[i]` (no conversion) |

When `Variadic` is true, extra arguments beyond `Args` are passed as
`...interface{}` to the method. The method should accept `extra ...interface{}`
as its last parameter.

**Type** — The Go struct type name that acts as the method receiver. A
package-level pointer instance (`var _<name> = &Type{}`) is generated
automatically, so methods use pointer receivers and can mutate struct fields.
By convention, use PascalCase: `os` → `OS`, `http` → `HTTP`, `conv` → `Conv`.

**GoImports** — Go import paths needed by the Runtime code beyond the base set.
The base set (`fmt`, `os`, `os/exec`, `strings`) is always available. Aliased
imports are supported: `gosimpleslug "github.com/gosimple/slug"` (entries
containing `"` are emitted verbatim).

**GoDeps** — Go module dependencies (require lines for go.mod) that the
generated program needs. Each entry should be `"module version"`, e.g.
`"github.com/gosimple/slug v1.15.0"`. Only needed for external modules that
depend on third-party Go packages not in the Go standard library.

**Runtime** — Go source containing the struct type definition and its methods,
embedded from `runtime.go`. Use `//go:embed` and `modules.CleanRuntime()` to
strip the package declaration, imports, and build tags before emission. Can
include `var _ = pkg.Symbol` lines to silence unused import warnings.

## Registry API

The `modules` package exposes these functions for use by the compiler:

```go
modules.Register(m *Module)                            // Add a module
modules.Get(name string) (*Module, bool)               // Lookup by name
modules.IsModule(name string) bool                     // Check if registered
modules.LookupFunc(module, func string) (string, bool) // Resolve module.func → wrapper name
modules.Names() []string                               // All registered names (sorted)
modules.CollectGoDeps(names []string) []string         // GoDeps from named modules
modules.CleanRuntime(src string) string                // Strip //go:build + package header
m.FullRuntime() string                                 // Impl code + generated wrappers
```

## Directory Layout

```
modules/
  module.go          # Module type + registry API + CleanRuntime helper
  module_test.go     # Registry and FullRuntime tests
  os/
    os.go            # Registration (embeds runtime.go)
    runtime.go       # Go runtime code (struct + methods)
  http/
    http.go          # Registration (embeds runtime.go)
    runtime.go       # Go runtime code (struct + methods)
    stubs.go         # Runtime helper stubs
  conv/
    conv.go          # Registration (embeds runtime.go)
    runtime.go       # Go runtime code (struct + methods)
    stubs.go         # Runtime helper stubs
  cli/
    cli.go           # Registration with DispatchEntry (embeds runtime.go)
    runtime.go       # CLI builder, parser, help, dispatch runner
    stubs.go         # Runtime helper stubs + dispatch var
    cli_test.go      # Unit tests
  json/
    json.go          # Registration (embeds runtime.go)
    runtime.go       # JSON parse/encode with Rugo type conversion
```
