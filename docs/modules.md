# Modules

Rugo has three module systems:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Load Rugo stdlib modules | `use "http"` |
| `import` | Bridge to Go stdlib packages | `import "strings"` |
| `require` | Load user `.rg` files | `require "helpers"` |

All three are accessed as `namespace.function()`.

## Rugo Stdlib (`use`)

Load built-in Rugo modules with `use`:

```ruby
use "http"
resp = http.get("https://example.com")
puts resp.body
```

| Module | Description |
|--------|-------------|
| [cli](modules/cli.md) | CLI app builder with commands, flags, and dispatch |
| [color](modules/color.md) | ANSI terminal colors and styles |
| [conv](modules/conv.md) | Type conversions |
| [fmt](modules/fmt.md) | String formatting (sprintf, printf) |
| [http](modules/http.md) | HTTP client |
| [json](modules/json.md) | JSON parsing and encoding |
| [os](modules/os.md) | Shell execution and process control |
| [re](modules/re.md) | Regular expressions |
| [str](modules/str.md) | String utilities |
| [test](modules/test.md) | Testing and assertions |

## Go Stdlib Bridge (`import`)

Access Go standard library packages directly with `import`:

```ruby
import "strings"
import "math"
import "strconv"

puts strings.contains("hello world", "world")  # true
puts math.sqrt(144.0)                           # 12
n = strconv.atoi("42")                          # 42
```

Function names are automatically converted from Rugo's `snake_case` to Go's
`PascalCase` (e.g., `strings.has_prefix` → `strings.HasPrefix`).

Use `as` to alias an import:

```ruby
import "strings" as str_go
puts str_go.to_upper("hello")   # HELLO
```

### Available Go Packages

| Package | Functions |
|---------|-----------|
| `strings` | contains, has_prefix, has_suffix, to_upper, to_lower, trim_space, repeat, replace, replace_all, split, join, index, count, trim, trim_left, trim_right, trim_prefix, trim_suffix, has_prefix, has_suffix, equal_fold, fields |
| `strconv` | atoi, itoa, format_float, parse_float, format_bool, parse_bool, format_int, parse_int |
| `math` | abs, ceil, floor, round, sqrt, pow, log, log2, log10, max, min, mod, sin, cos, tan |
| `math/rand/v2` | int_n, float64, n |
| `path/filepath` | join, base, dir, ext, clean, is_abs, rel, split |
| `sort` | strings (sorts a string array), ints (sorts an int array) |
| `os` | getenv, setenv, read_file, write_file, mkdir_all, remove, remove_all, getwd |
| `time` | now_unix, now_nano, sleep |

### Error Handling

Go functions that return `(T, error)` auto-panic on error, integrating with `try/or`:

```ruby
import "strconv"
n = try strconv.atoi("abc") or 0   # returns 0 on error
```

## Builtins

Available without any import:

| Function | Description |
|----------|-------------|
| `puts(args...)` | Print with newline |
| `print(args...)` | Print without newline |
| `len(collection)` | Length of array, hash, or string |
| `append(array, item)` | Append item, returns new array |

## User Modules (`require`)

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

Functions are namespaced by filename. Use `as` to pick a custom namespace:

```ruby
require "math_helpers" as "m"
puts m.double(21)   # 42
```

### Remote Modules

Load `.rg` modules directly from git repositories:

```ruby
require "github.com/user/rugo-utils@v1.0.0" as "utils"
puts utils.slugify("Hello World")
```

| Syntax | Meaning |
|--------|---------|
| `@v1.2.0` | Git tag (cached forever) |
| `@main` | Branch (re-fetched each build) |
| `@abc1234` | Commit SHA (cached forever) |
| *(none)* | Default branch (re-fetched) |

Remote modules are cached in `~/.rugo/modules/`. Tagged versions and commit
SHAs are immutable — once downloaded, they're never re-fetched.

### Search Path

Rugo resolves `require` paths with two simple rules:

1. **Relative to calling file** — `require "helpers"` loads `helpers.rg` from
   the same directory as the file containing the `require`. Subdirectories
   work too: `require "lib/utils"` loads `lib/utils.rg`. The `.rg` extension
   is added automatically if missing.

2. **Remote URL** — if the path looks like a URL (`github.com/user/repo`),
   Rugo fetches the repository via git and resolves the entry point.

There is no `$RUGO_PATH`, no implicit search directories, and no walking up
the directory tree. The require string tells you exactly where the code comes
from.

## Namespace Rules

- `use` and `require` must be at the top level (not inside `def`, `if`, etc.)
- A namespace can only be claimed once — if `use "os"` is loaded, `import "os"` must be aliased: `import "os" as go_os`
- Each module can only be imported/used once
