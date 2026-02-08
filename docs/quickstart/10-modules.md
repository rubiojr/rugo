# Modules

Rugo has two module systems: **stdlib modules** are built-in Go modules loaded with `import`, while **user modules** are `.rg` files you write and load with `require`. Both are accessed as `module.function()`.

## Stdlib Modules

Import with `import`, call as `module.function`:

```ruby
import "http"
import "os"
import "conv"
import "str"
```

Quick examples:

```ruby
body = http.get("https://example.com")
hostname = `hostname`
n = conv.to_i("42")
parts = str.split("a,b,c", ",")
```

See the full [Modules Reference](../modules.md) for all functions.

## Global Builtins

Available without import: `puts`, `print`, `len`, `append`.

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

Functions are namespaced by filename.

---
Next: [Error Handling](11-error-handling.md)
