# Go Bridge

Rugo can call Go standard library functions directly using `import`. No wrapper
modules needed — the compiler auto-generates type conversions.

## Basic Usage

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")                  # HELLO
puts strings.contains("hello world", "world")   # true
puts math.sqrt(144.0)                           # 12
puts math.pow(2.0, 10.0)                        # 1024
```

Function names use `snake_case` in Rugo — they're auto-converted to Go's
`PascalCase` (`to_upper` → `ToUpper`, `has_prefix` → `HasPrefix`).

## Type Conversions

```ruby
import "strconv"

n = strconv.atoi("42")           # string → int
s = strconv.itoa(42)             # int → string
f = strconv.parse_float("3.14")  # string → float
```

## Error Handling

Go functions that return `(T, error)` auto-panic on error. Use `try/or`:

```ruby
import "strconv"

# Panics on invalid input
n = strconv.atoi("42")

# Recover with try/or
n = try strconv.atoi("not a number") or 0
puts n   # 0
```

## Aliasing

Use `as` to alias an import — useful when a Go package name conflicts
with a Rugo stdlib module:

```ruby
use "os"                    # Rugo os module (exec, exit)
import "os" as go_os        # Go os package (getenv, read_file)

go_os.setenv("APP", "rugo")
puts go_os.getenv("APP")    # rugo
```

## Available Packages

| Package | Key Functions |
|---------|--------------|
| `strings` | contains, has_prefix, has_suffix, to_upper, to_lower, trim_space, split, join, replace, repeat, index, count, fields |
| `strconv` | atoi, itoa, format_float, parse_float, format_bool, parse_bool |
| `math` | abs, ceil, floor, round, sqrt, pow, log, max, min, sin, cos, tan |
| `path/filepath` | join, base, dir, ext, clean, is_abs, rel, split |
| `regexp` | match_string, must_compile, compile |
| `sort` | strings, ints |
| `os` | getenv, setenv, read_file, write_file, mkdir_all, remove, getwd |
| `time` | now_unix, now_nano, sleep |

## `use` vs `import`

| | `use` | `import` |
|--|-------|----------|
| **Purpose** | Rugo stdlib modules | Go stdlib packages |
| **Implementation** | Hand-crafted Go wrappers | Auto-generated bridge |
| **Example** | `use "http"` | `import "strings"` |
| **Functions** | Rugo-native API | Go function names in snake_case |

When both exist for the same name (e.g., `os`), alias the Go import:
`import "os" as go_os`.

---
See the full [Modules Reference](../modules.md) for all available functions.
