# Modules

Rugo has three module systems:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Load Rugo stdlib modules | `use "http"` |
| `import` | Bridge to Go stdlib packages | `import "strings"` |
| `require` | Load user `.rugo` files | `require "helpers"` |

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
| [eval](modules/eval.md) | Compile and run Rugo code at runtime |
| [fmt](modules/fmt.md) | String formatting (sprintf, printf) |
| [http](modules/http.md) | HTTP client |
| [json](modules/json.md) | JSON parsing and encoding |
| [os](modules/os.md) | Shell execution and process control |
| [queue](modules/queue.md) | Thread-safe queue for producer-consumer concurrency |
| [re](modules/re.md) | Regular expressions |
| [sqlite](modules/sqlite.md) | SQLite database access |
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
| `encoding/json` | marshal, unmarshal, marshal_indent |
| `encoding/base64` | encode, decode, url_encode, url_decode |
| `encoding/hex` | encode, decode |
| `crypto/sha256` | sum256 |
| `crypto/md5` | sum |
| `net/url` | parse, path_escape, path_unescape, query_escape, query_unescape |
| `unicode` | is_letter, is_digit, is_space, is_upper, is_lower, is_punct, to_upper, to_lower |
| `slices` | contains, index, reverse, compact |
| `maps` | keys, values, clone, equal |

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
| `append(array, item)` | Append item to array. Bare form: `append arr, item` |

## User Modules (`require`)

Create reusable `.rugo` files and load them with `require`:

```ruby
# math_helpers.rugo
def double(n)
  return n * 2
end
```

```ruby
# main.rugo
require "math_helpers"
puts math_helpers.double(21)   # 42
```

Functions are namespaced by filename. Use `as` to pick a custom namespace:

```ruby
require "math_helpers" as "m"
puts m.double(21)   # 42
```

### Remote Modules

Load `.rugo` modules directly from git repositories:

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

#### Selective Imports with `with`

Load specific `.rugo` files from a local directory or remote repository using `with`:

```ruby
# Local directory
require "mylib" with client, helpers
client.make("token")

# Remote repository
require "github.com/user/my-lib@v1.0.0" with client, issue
client.make("token")
issue.list(gh, "owner", "repo")
```

Each name loads `<name>.rugo` from the directory or repo root as its own namespace.
If not found at the root, `lib/<name>.rugo` is checked as a fallback.

For remote modules, without `with`, the repo's entry point (`main.rugo`, `<repo-name>.rugo`, or the
sole `.rugo` file) is loaded. With `with`, the entry point is bypassed and each
named file is loaded directly.

`with` and `as` are mutually exclusive. For local requires, the path must be a directory.

#### Subpath Requires

You can also require a specific file from a remote repo by path:

```ruby
require "github.com/user/my-lib/client@v1.0.0"
# loads client.rugo from the repo root, namespace "client"
```

### Search Path

Rugo resolves `require` paths with two simple rules:

1. **Relative to calling file** — `require "helpers"` loads `helpers.rugo` from
   the same directory as the file containing the `require`. Subdirectories
   work too: `require "lib/utils"` loads `lib/utils.rugo`. The `.rugo` extension
   is added automatically if missing. If the path resolves to a directory,
   Rugo looks for an entry point: `<dirname>.rugo` → `main.rugo` → sole `.rugo`
   file. A file always takes precedence over a directory of the same name.

2. **Remote URL** — if the path looks like a URL (`github.com/user/repo`),
   Rugo fetches the repository via git and resolves the entry point.

There is no `$RUGO_PATH`, no implicit search directories, and no walking up
the directory tree. The require string tells you exactly where the code comes
from.

## Namespace Rules

- `use` and `require` must be at the top level (not inside `def`, `if`, etc.)
- A namespace can only be claimed once — if `use "os"` is loaded, `import "os"` must be aliased: `import "os" as go_os`
- Each module can only be imported/used once
