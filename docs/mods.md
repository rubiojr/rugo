# Rugo Module System

Rugo uses a registry-based module system. Each stdlib module lives in its own
directory under `modules/` and self-registers at startup via Go's `init()`.

## Built-in Modules

| Module | Functions | Description |
|--------|-----------|-------------|
| `os`   | `exec`, `exit` | Shell execution and process control |
| `http` | `get`, `post` | HTTP client |
| `conv` | `to_i`, `to_f`, `to_s` | Type conversions |

### Usage in Rugo

```ruby
import "http"
import "conv"

result = `whoami`
puts result

body = http.get("https://example.com")
puts conv.to_s(42)
```

## Creating a New Module

### 1. Create the module directory

```
modules/mymod/
  mymod.go       # Registration
  runtime.go     # Runtime Go code (//go:build ignore)
```

### 2. Write the runtime as typed Go code

`modules/mymod/runtime.go`:

```go
//go:build ignore

package mymod

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

The `//go:build ignore` tag excludes this file from compilation — it's only
embedded as a string. The `package` declaration lets editors provide syntax
highlighting. Both are stripped by `CleanRuntime` before emission.

Runtime functions can call base runtime helpers (`rugo_to_string`, `rugo_to_int`,
`rugo_to_float`, `rugo_to_bool`) and any Go stdlib package from the base import
set (`fmt`, `os`, `os/exec`, `strings`).

### 3. Register the module with typed function definitions

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

### 4. Wire it up in `main.go`

Add a blank import so the module's `init()` runs:

```go
import (
    _ "github.com/rubiojr/rugo/modules/mymod"
)
```

### 5. Use it in Rugo

```ruby
import "mymod"

puts mymod.hello("developer")
```

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
    Name      string      // Rugo import name (e.g. "os")
    Type      string      // Go struct type name (e.g. "OS", "HTTP")
    Funcs     []FuncDef   // Function definitions with typed args
    GoImports []string    // Additional Go imports (e.g. ["net/http"])
    Runtime   string      // Struct type + methods from runtime.go
}
```

### Field details

**Name** — The string used in `import "name"` in Rugo source files.

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
The base set (`fmt`, `os`, `os/exec`, `strings`) is always available.

**Runtime** — Go source containing the struct type definition and its methods,
embedded from `runtime.go`. Use `//go:embed` and `modules.CleanRuntime()` to
strip the build tag and package header. Can include `var _ = pkg.Symbol` lines
to silence unused import warnings.

## Registry API

The `modules` package exposes these functions for use by the compiler:

```go
modules.Register(m *Module)                            // Add a module
modules.Get(name string) (*Module, bool)               // Lookup by name
modules.IsModule(name string) bool                     // Check if registered
modules.LookupFunc(module, func string) (string, bool) // Resolve module.func → wrapper name
modules.Names() []string                               // All registered names (sorted)
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
    runtime.go       # Go runtime code (//go:build ignore)
  http/
    http.go          # Registration (embeds runtime.go)
    runtime.go       # Go runtime code (//go:build ignore)
  conv/
    conv.go          # Registration (embeds runtime.go)
    runtime.go       # Go runtime code (//go:build ignore)
```
