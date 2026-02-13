---
name: rugo-go-modules
description: Expert in writing and using Go modules from Rugo via `require`. Load when creating Go packages that bridge to Rugo, debugging Go module imports, or helping users expose Go functions to Rugo scripts.
---

# Rugo Go Modules

Bridge Go packages into Rugo via `require` — the compiler introspects Go source, discovers exported functions, and bridges them automatically. No manifest, no registration boilerplate, no custom binary needed.

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

No `init()`, no `modules.Register()`, no `runtime.go`. Just standard Go code with exported functions.

## Using from Rugo

```ruby
require "greeter"
use "conv"

puts(greeter.hello("World"))              # Hello, World!
puts(greeter.shout("hello"))              # HELLO!
puts(conv.to_s(greeter.add(3, 4)))        # 7
puts(conv.to_s(greeter.is_empty("")))     # true
```

Function names are automatically converted from PascalCase to snake_case: `Hello` → `hello`, `IsEmpty` → `is_empty`.

## Module Naming

The Rugo namespace is derived from the last segment of the require path:

| Require | Namespace | Derived from |
|---------|-----------|--------------|
| `require "greeter"` | `greeter` | directory name |
| `require "path/to/mylib"` | `mylib` | last path segment |
| `require "github.com/user/rugo-slug@v1"` | `rugo_slug` | repo name (hyphens → underscores) |
| `require "greeter" as g` | `g` | explicit alias |

The Go package name and `go.mod` module path don't affect the namespace. Hyphens are converted to underscores. Use `as` to override.

## Namespace Aliasing

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

Go module requires work with `rugo build` — the generated binary includes the Go module code:

```bash
rugo build main.rugo -o myapp
./myapp   # works without the Go module source at runtime
```

## Remote Go Modules

Go modules can be hosted in git repositories and required by URL:

```ruby
require "github.com/user/rugo-greeter@v1.0.0" as greeter
puts(greeter.hello("Remote"))
```

Remote modules are cached in `~/.rugo/modules/` and locked via `rugo.lock`. `rugo doc` works on remote Go modules too.

## Supported Types

Exported functions must use bridgeable types:

| Go type    | Rugo type | Notes |
|------------|-----------|-------|
| `string`   | string    | |
| `int`      | integer   | |
| `float64`  | float     | |
| `bool`     | boolean   | |
| `error`    | —         | auto-panics on non-nil |
| `[]string` | array     | |
| `[]byte`   | string    | cast |

Functions with non-bridgeable types (pointers, interfaces, channels, maps, structs, generics) are automatically excluded. The compiler emits a warning for each skipped function:

```
warning: mymod: skipping Fail() — param 0: pointer to mymod.Foo
```

If no functions are bridgeable, the compiler reports an error listing each function and why it was blocked.

## Limitations

- **Only exported package-level functions are bridged.** Methods on structs, struct types, variables, and constants are not exposed. For stateful objects, use a custom build instead.
- **Only top-level `.go` files are inspected.** Sub-packages in subdirectories are not automatically included. Use the `with` clause or require sub-packages directly:
  ```ruby
  require "mymod" with utils       # inspects mymod/utils/
  require "mymod/utils"            # also works
  require "mymod/utils" as u       # with alias
  ```
- **Unexported functions are ignored** — only `func Uppercase(...)` style exports are discovered.

## Tips

- **Use Go stdlib freely** — your Go code can import any Go standard library package. Only the exported function signatures need bridgeable types.
- **External dependencies** — your `go.mod` can declare `require` entries for third-party Go packages. The compiler resolves them automatically via `go build -mod=mod`.
- **PascalCase → snake_case** — common abbreviations are handled: `URL` → `url`, `JSON` → `json`, `IsNaN` → `is_nan`.

## When to Use This vs Custom Builds

| | Go modules via `require` | Custom builds (`use`) |
|---|---|---|
| **Setup** | None — just write Go | Build custom binary |
| **State** | Stateless (package-level funcs) | Stateful (struct with fields) |
| **Types** | Bridgeable types only | Any Go type |
| **Dispatch** | No | Yes (CLI/web handlers) |
| **Dependencies** | Automatic (from go.mod) | Via GoDeps field |

Use `require` for wrapping Go libraries. Use custom builds when you need stateful modules, dispatch, or complex type handling.

## Working Example

See `examples/require_go/` in the Rugo repository.
