# Modules

Import stdlib modules with `import`, call as `module.function`:

```ruby
import "http"
body = http.get("https://example.com")
```

## Stdlib

| Module | Description |
|--------|-------------|
| [cli](modules/cli.md) | CLI app builder with commands, flags, and dispatch |
| [color](modules/color.md) | ANSI terminal colors and styles |
| [http](modules/http.md) | HTTP client |
| [os](modules/os.md) | Shell execution and process control |
| [conv](modules/conv.md) | Type conversions |
| [json](modules/json.md) | JSON parsing and encoding |
| [str](modules/str.md) | String utilities |
| [test](modules/test.md) | Testing and assertions |

## Builtins

Available without import:

| Function | Description |
|----------|-------------|
| `puts(args...)` | Print with newline |
| `print(args...)` | Print without newline |
| `len(collection)` | Length of array, hash, or string |
| `append(array, item)` | Append item, returns new array |

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
