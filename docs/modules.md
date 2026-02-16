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
| [ast](modules/ast.md) | Parse and inspect Rugo source files |
| [base64](#base64) | Base64 encoding and decoding |
| [cli](modules/cli.md) | CLI app builder with commands, flags, and dispatch |
| [color](modules/color.md) | ANSI terminal colors and styles |
| [conv](modules/conv.md) | Type conversions (to_i, to_f, to_s, to_bool, parse_int) |
| [crypto](#crypto) | Cryptographic hash functions (md5, sha1, sha256) |
| [eval](modules/eval.md) | Compile and run Rugo code at runtime |
| [filepath](modules/filepath.md) | File path manipulation and querying |
| [fmt](modules/fmt.md) | String formatting (sprintf, printf) |
| [hex](#hex) | Hex encoding and decoding |
| [http](modules/http.md) | HTTP client |
| [json](modules/json.md) | JSON parsing, encoding, and pretty-printing |
| [math](modules/math.md) | Mathematical functions and constants |
| [os](modules/os.md) | Shell execution, process control, and filesystem operations |
| [queue](modules/queue.md) | Thread-safe queue for producer-consumer concurrency |
| [rand](#rand) | Random number generation, shuffling, and UUIDs |
| [re](modules/re.md) | Regular expressions |
| [sqlite](modules/sqlite.md) | SQLite database access |
| [str](modules/str.md) | String utilities |
| [test](modules/test.md) | Testing and assertions |
| [time](modules/time.md) | Time operations: timestamps, sleeping, formatting, and parsing |

### Module Quick Reference

#### str

String utilities: contains, split, trim, starts_with, ends_with, replace, upper, lower, index, join, rune_count, count, repeat, reverse, chars, fields, trim_prefix, trim_suffix, pad_left, pad_right, each_line, center, last_index, slice, empty.

```ruby
use "str"
puts str.reverse("hello")            # olleh
puts str.pad_left("42", 6, "0")      # 000042
puts str.center("title", 20, "-")    # -------title--------
puts str.count("banana", "a")        # 3
puts str.empty("")                   # true
```

#### json

JSON parsing, encoding, and pretty-printing: parse, encode, pretty.

```ruby
use "json"
data = {"name": "Rugo", "version": 1}
puts json.pretty(data)
```

#### os

Shell execution, process control, and filesystem operations: exec, exit, file_exists, is_dir, read_line, getenv, setenv, cwd, chdir, hostname, read_file, write_file, remove, mkdir, rename, glob, tmp_dir, args, pid, symlink, readlink.

```ruby
use "os"
puts os.cwd()
puts os.hostname()
puts os.getenv("HOME")
```

#### conv

Type conversions: to_i, to_f, to_s, to_bool, parse_int.

```ruby
use "conv"
puts conv.to_bool("true")    # true
puts conv.parse_int("ff", 16) # 255
```

#### math

Mathematical functions and constants: abs, ceil, floor, round, max, min, pow, sqrt, log, log2, log10, sin, cos, tan, pi, e, inf, nan, is_nan, is_inf, clamp, random, random_int.

```ruby
use "math"
puts math.sqrt(144.0)        # 12
puts math.pi()                # 3.141592653589793
puts math.clamp(15, 0, 10)   # 10
```

#### filepath

File path manipulation: join, base, dir, ext, abs, rel, glob, clean, is_abs, split, match.

```ruby
use "filepath"
puts filepath.join("home", "user", "docs")   # home/user/docs
puts filepath.ext("photo.jpg")                # .jpg
puts filepath.base("/home/user/file.txt")     # file.txt
```

#### time

Time operations: now, sleep, format, parse, since, millis.

```ruby
use "time"
t = time.now()
time.sleep(100)               # sleep 100ms
puts time.millis()            # current time in milliseconds
```

#### base64

Base64 encoding and decoding: encode, decode, url_encode, url_decode.

```ruby
use "base64"
encoded = base64.encode("Hello, Rugo!")
puts encoded                               # SGVsbG8sIFJ1Z28h
puts base64.decode(encoded)                # Hello, Rugo!
```

#### hex

Hex encoding and decoding: encode, decode.

```ruby
use "hex"
puts hex.encode("hello")     # 68656c6c6f
puts hex.decode("68656c6c6f") # hello
```

#### crypto

Cryptographic hash functions: md5, sha256, sha1.

```ruby
use "crypto"
puts crypto.sha256("hello")
puts crypto.md5("hello")
```

#### rand

Random number generation, shuffling, and UUIDs: int, float, string, choice, shuffle, uuid.

```ruby
use "rand"
puts rand.uuid()
puts rand.int(1, 100)
puts rand.choice(["a", "b", "c"])
```

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

## Module Visibility

Functions prefixed with `_` are private to their module. The compiler rejects
calls to `_`-prefixed functions from outside the module:

```ruby
# helpers.rugo
def _internal()
  return "secret"
end

def public()
  return _internal()   # OK: same module
end
```

```ruby
# main.rugo
require "helpers"
puts helpers.public()      # OK
puts helpers._internal()   # compile error: '_internal' is private to module 'helpers'
```

This applies to all `require` forms (`as`, `with`) and to struct methods
(`def Dog._validate()`).
