# Go Modules via Require (Advanced)

Rugo can `require` Go packages directly — the compiler introspects the Go
source, discovers exported functions, and bridges them automatically. No
manifest, no registration boilerplate, no custom binary needed.

## Creating a Go Module

Create a directory with a `go.mod` and a `.go` file:

```
greeter/
  go.mod
  greeter.go
```

**go.mod:**

```
module example.com/greeter

go 1.22
```

**greeter.go:**

```go
package greeter

import "strings"

func Hello(name string) string {
    return "Hello, " + name + "!"
}

func Shout(text string) string {
    return strings.ToUpper(text) + "!"
}

func Add(a int, b int) int {
    return a + b
}

func IsEmpty(s string) bool {
    return len(s) == 0
}
```

That's it — no `init()`, no `modules.Register()`, no `runtime.go`. Just
standard Go code with exported functions.

## Using It from Rugo

```ruby
require "greeter"
use "conv"

puts(greeter.hello("World"))              # Hello, World!
puts(greeter.shout("hello"))              # HELLO!
puts(conv.to_s(greeter.add(3, 4)))        # 7
puts(conv.to_s(greeter.is_empty("")))     # true
```

Function names are automatically converted from PascalCase to snake_case:
`Hello` → `hello`, `IsEmpty` → `is_empty`.

## Module Naming

The Rugo namespace is derived from **how you require the module** — the
last segment of the require path, same as Rugo modules:

| Require | Namespace | Derived from |
|---------|-----------|--------------|
| `require "greeter"` | `greeter` | directory name |
| `require "path/to/mylib"` | `mylib` | last path segment |
| `require "github.com/user/rugo-slug@v1"` | `rugo_slug` | repo name (hyphens → underscores) |
| `require "greeter" as g` | `g` | explicit alias |

The Go package name and `go.mod` module path don't affect the namespace.
Hyphens are converted to underscores. Use `as` to override.

## Namespace Aliasing

Use `as` to pick a shorter namespace:

```ruby
require "greeter" as g

puts(g.hello("Alias"))   # Hello, Alias!
```

## Viewing Documentation

`rugo doc` works on Go module directories:

```
$ rugo doc greeter

package greeter (Go: example.com/greeter)
    Functions from Go module example.com/greeter.

greeter.add(int, int) -> int
greeter.hello(string) -> string
greeter.is_empty(string) -> bool
greeter.shout(string) -> string
```

## Building Binaries

Go module requires work with `rugo build` — the generated binary includes
the Go module code:

```bash
rugo build main.rugo -o myapp
./myapp   # works without the Go module source at runtime
```

## Remote Go Modules

Go modules can be hosted in git repositories and required by URL, just like
remote Rugo modules:

```ruby
require "github.com/user/rugo-greeter@v1.0.0" as greeter
puts(greeter.hello("Remote"))
```

Remote modules are cached in `~/.rugo/modules/` and locked via `rugo.lock`.
`rugo doc` works on remote Go modules too:

```
$ rugo doc github.com/user/rugo-greeter@v1.0.0
```

## Supported Types

Exported functions must use types that can be bridged to Rugo:

| Go type    | Rugo type | Notes |
|------------|-----------|-------|
| `string`   | string    | |
| `int`      | integer   | |
| `float64`  | float     | |
| `bool`     | boolean   | |
| `error`    | —         | auto-panics on non-nil |
| `[]string` | array     | |
| `[]byte`   | string    | cast |
| `*Struct`  | opaque handle | field get/set via dot syntax |

Functions with non-bridgeable types (interfaces, channels, maps, generics)
are automatically excluded. The compiler emits a warning for each skipped
function so module authors know what's not available:

```
warning: mymod: skipping Fail() — param 0: interface type
```

If no functions or structs are bridgeable, the compiler reports an error
listing each function and why it was blocked.

## Struct Support

Exported structs with bridgeable field types are automatically discovered.
The compiler generates a zero-value constructor and bridges functions that
take or return struct pointers.

**Go module:**

```go
package mymod

type Config struct {
    Name string
    Port int
}

func NewConfig(name string, port int) *Config {
    return &Config{Name: name, Port: port}
}

func Describe(c *Config) string {
    return c.Name + " server"
}
```

**Rugo script:**

```ruby
require "mymod"

# Zero-value constructor (snake_case of struct name)
c = mymod.config()
c.name = "app"
c.port = 8080

# Go constructor that returns *Config
c2 = mymod.new_config("app", 8080)

# Pass struct to Go function
puts(mymod.describe(c2))   # app server

# Field access
puts(c2.name)              # app
puts(c2.port)              # 8080

# Type introspection
puts(type_of(c2))          # Config
```

### What works

- Exported structs with `string`, `int`, `float64`, `bool` fields
- Zero-value constructors: `mymod.config()` → `&Config{}`
- `New*` constructors: `NewConfig(...)` → `mymod.new_config(...)`
- Functions taking `*Struct` params
- Functions returning `*Struct`
- Field get (`c.name`) and set (`c.name = "x"`)
- `type_of(c)` returns the Go struct name

### Limitations

- Struct methods (`c.describe()`) are not bridged — use package-level functions
  that take the struct as a parameter instead
- Nested structs (struct fields that are other structs) are skipped
- Struct fields with non-bridgeable types are skipped (the struct is still
  bridgeable with the remaining fields)

## Limitations

- **Only exported package-level functions are bridged.** Methods on structs
  are not exposed. Use package-level functions that take struct pointers instead.
- **Only top-level `.go` files are inspected.** Sub-packages in subdirectories
  are not automatically included. Use the `with` clause or require sub-packages
  directly:
  ```ruby
  require "mymod" with utils       # inspects mymod/utils/
  require "mymod/utils"            # also works
  require "mymod/utils" as u       # with alias
  ```
- **Unexported functions are ignored** — only `func Uppercase(...)` style
  exports are discovered.

## Tips

- **Use Go stdlib freely** — your Go code can import any Go standard library
  package. Only the exported function signatures need bridgeable types.
- **External dependencies** — your `go.mod` can declare `require` entries
  for third-party Go packages. The compiler resolves them automatically
  via `go build -mod=mod`.
- **PascalCase → snake_case** — common abbreviations are handled:
  `URL` → `url`, `JSON` → `json`, `IsNaN` → `is_nan`.

## When to Use This vs Custom Builds

| | Go modules via `require` | Custom builds (`use`) |
|---|---|---|
| **Setup** | None — just write Go | Build custom binary |
| **State** | Structs with field access | Full struct methods |
| **Types** | Bridgeable types only | Any Go type |
| **Dispatch** | No | Yes (CLI/web handlers) |
| **Dependencies** | Automatic (from go.mod) | Via GoDeps field |

Use `require` for wrapping Go libraries. Use [custom builds](14-custom-modules.md)
when you need struct methods, dispatch, or complex type handling.

---

See `examples/require_go/` and `examples/require_go_struct/` for working examples.
