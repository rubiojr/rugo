# Modules

Rugo has three module systems:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules | `use "http"` |
| `import` | Go stdlib bridge | `import "strings"` |
| `require` | User `.rg` files | `require "helpers"` |

## Rugo Stdlib Modules

Load with `use`, call as `module.function`:

```ruby
use "http"
use "conv"
use "str"
```

Quick examples:

```ruby
body = http.get("https://example.com")
n = conv.to_i("42")
parts = str.split("a,b,c", ",")
```

## Go Stdlib Bridge

Access Go standard library packages directly with `import`:

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")   # HELLO
puts math.sqrt(144.0)            # 12
```

Function names use `snake_case` in Rugo and are auto-converted to Go's `PascalCase`.

Use `as` to alias: `import "strings" as str_go`.

See the full [Modules Reference](../modules.md) for all available packages and functions.

## Global Builtins

Available without any import: `puts`, `print`, `len`, `append`.

## User Modules

Create reusable `.rg` files and load them with `require`:

```ruby
# math_helpers.rg
def double(n)
  return n * 2
end
```

```ruby
# main.rg
require "math_helpers"
puts math_helpers.double(21)   # 42
```

Functions are namespaced by filename. User modules can `use` Rugo stdlib modules in their functions — the imports are automatically propagated.

Paths are resolved relative to the calling file. `require "lib/utils"` loads `lib/utils.rg` from the calling file's directory.

### Remote Modules

Load modules directly from git repositories:

```ruby
require "github.com/user/rugo-utils@v1.0.0" as "utils"
puts utils.hello("world")
```

Pin a version with `@v1.0.0` (git tag) or `@main` (branch). Remote modules are cached in `~/.rugo/modules/`.

**Rules:**
- `use`, `import`, and `require` must be at the top level (not inside `def`, `if`, etc.)
- Namespaces must be unique — if `use "os"` is loaded, alias the Go bridge: `import "os" as go_os`
- Each module can only be imported/used once

---
Next: [Error Handling](11-error-handling.md)
