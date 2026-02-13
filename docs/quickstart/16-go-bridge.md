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

## Multi-Return Functions

Go functions that return multiple values are bridged as arrays. Use
destructuring to unpack them:

```ruby
import "strings"

before, after, found = strings.cut("key=value", "=")
puts before   # key
puts after    # value
puts found    # true
```

You can also access elements by index:

```ruby
result = strings.cut("hello-world", "-")
puts result[0]   # hello
puts result[1]   # world
```

## Available Packages

| Package | Key Functions |
|---------|--------------|
| `strings` | contains, has_prefix, has_suffix, to_upper, to_lower, trim_space, split, join, replace, repeat, index, count, fields, contains_func, index_func, map |
| `strconv` | atoi, itoa, format_float, parse_float, format_bool, parse_bool |
| `math` | abs, ceil, floor, round, sqrt, pow, log, max, min, sin, cos, tan |
| `path` | base, clean, dir, ext, is_abs, join, match, split |
| `path/filepath` | join, base, dir, ext, clean, is_abs, rel, split |
| `sort` | strings, ints |
| `os` | getenv, setenv, read_file, write_file, mkdir_all, remove, getwd |
| `time` | now_unix, now_nano, sleep |
| `encoding/json` | marshal, unmarshal, marshal_indent |
| `encoding/base64` | encode, decode, url_encode, url_decode |
| `encoding/hex` | encode, decode |
| `crypto/sha256` | sum256 |
| `crypto/md5` | sum |
| `net/url` | parse, path_escape, path_unescape, query_escape, query_unescape |
| `unicode` | is_letter, is_digit, is_space, is_upper, is_lower, is_punct, to_upper, to_lower |
| `html` | escape_string, unescape_string |
| `slices` | contains, index, reverse, compact |
| `maps` | keys, values, clone, equal |

## JSON

```ruby
import "encoding/json"

data = {name: "Rugo", version: 1}
text = json.marshal(data)
puts text                        # {"name":"Rugo","version":1}

parsed = json.unmarshal(text)
puts parsed.name                 # Rugo

puts json.marshal_indent(data, "", "  ")  # pretty-printed
```

## Encoding (Base64 & Hex)

```ruby
import "encoding/base64"
import "encoding/hex"

b64 = base64.encode("Hello!")
puts base64.decode(b64)          # Hello!

h = hex.encode("Hello!")
puts hex.decode(h)               # Hello!
```

## Hashing

```ruby
import "crypto/sha256"
import "crypto/md5"

puts sha256.sum256("hello")      # hex-encoded SHA-256
puts md5.sum("hello")            # hex-encoded MD5
```

## URL Parsing

```ruby
import "net/url"

u = url.parse("https://example.com:8080/path?q=hello#top")
puts u.scheme     # https
puts u.hostname   # example.com
puts u.port       # 8080
puts u.path       # /path
puts u.query      # q=hello
puts u.fragment   # top

escaped = url.query_escape("hello world")
puts url.query_unescape(escaped)   # hello world
```

## Collections (Slices & Maps)

```ruby
import "slices"
import "maps"

puts slices.contains(["a", "b", "c"], "b")  # true
puts slices.reverse([1, 2, 3])               # [3, 2, 1]

h = {name: "Rugo", lang: "go"}
puts maps.keys(h)                # [lang, name]
copy = maps.clone(h)
puts maps.equal(h, copy)         # true
```

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
