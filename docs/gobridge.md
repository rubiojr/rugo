# Go Bridge — Developer Reference

The Go Bridge allows Rugo scripts to call Go standard library functions
directly via the `import` keyword. The compiler maintains a static registry
of whitelisted functions and auto-generates type-safe Go calls with
`interface{}` ↔ typed conversions.

## Architecture

```
Rugo source:    import "strings"
                x = strings.to_upper("hello")
                    │
                    ▼
Compiler:       resolveRequires() validates package is whitelisted
                    │
                    ▼
Codegen:        generateGoBridgeCall() produces:
                  interface{}(strings.ToUpper(rugo_to_string(x)))
                    │
                    ▼
Generated Go:   Direct Go stdlib call — no wrapper struct, no runtime
```

### Key files

| File | Role |
|------|------|
| `compiler/gobridge/gobridge.go` | Exported types (`GoType`, `GoFuncSig`, `Package`), registry API, conversion helpers |
| `compiler/gobridge/strings.go` | strings package mappings (self-registers via `init()`) |
| `compiler/gobridge/strconv.go` | strconv package mappings |
| `compiler/gobridge/math.go` | math package mappings |
| `compiler/gobridge/rand.go` | math/rand/v2 package mappings |
| `compiler/gobridge/filepath.go` | path/filepath package mappings |
| `compiler/gobridge/sort.go` | sort package mappings |
| `compiler/gobridge/os.go` | os package mappings |
| `compiler/gobridge/time.go` | time package mappings |
| `compiler/gobridge/json.go` | encoding/json package mappings |
| `compiler/gobridge/base64.go` | encoding/base64 package mappings |
| `compiler/gobridge/hex.go` | encoding/hex package mappings |
| `compiler/gobridge/crypto.go` | crypto/sha256 + crypto/md5 package mappings |
| `compiler/gobridge/url.go` | net/url package mappings |
| `compiler/gobridge/unicode.go` | unicode package mappings |
| `compiler/gobridge/slices.go` | slices package mappings (runtime helpers) |
| `compiler/gobridge/maps.go` | maps package mappings (runtime helpers) |
| `compiler/compiler.go` | `resolveRequires()` — validates + deduplicates `import` statements |
| `compiler/codegen.go` | `generateGoBridgeCall()` — emits Go code with type conversions |
| `compiler/nodes.go` | `ImportStmt` AST node (Package + Alias fields) |
| `compiler/walker.go` | `walkImportStmt()` — walks parse tree into `ImportStmt` |
| `compiler/preprocess.go` | `import` keyword registered to avoid shell fallback |
| `parser/rugo.ebnf` | Grammar: `ImportStmt = "import" str_lit ["as" ident] .` |

### How it differs from `use` (Rugo stdlib)

| | `use` (Rugo modules) | `import` (Go bridge) |
|--|---|---|
| **Mechanism** | Hand-crafted Go structs with method receivers, embedded as runtime strings | Static registry lookup → direct Go function calls |
| **Type safety** | Module defines `FuncDef.Args` → codegen wraps each arg | Registry defines `GoFuncSig.Params` → codegen converts each arg |
| **Generated code** | `rugo_mod_func(arg1, arg2)` wrapper | `strings.ToUpper(rugo_to_string(arg))` direct call |
| **Adding functions** | Write Go struct + methods in `runtime.go`, register in module | Add entry to mapping file in `compiler/gobridge/` |

## Type System

Rugo is dynamically typed (`interface{}`). The bridge converts at call
boundaries using the same runtime helpers the rest of the compiler uses:

```go
type GoType int

const (
    GoString      GoType = iota  // → rugo_to_string(arg)
    GoInt                         // → rugo_to_int(arg)
    GoFloat64                     // → rugo_to_float(arg)
    GoBool                        // → rugo_to_bool(arg)
    GoByte                        // → byte(rugo_to_int(arg))
    GoStringSlice                 // → rugo_go_to_string_slice(arg)
    GoError                       // Not a param type — return-only
)
```

Return conversions wrap Go values back to `interface{}`:
- Most types: `interface{}(goValue)`
- `GoStringSlice`: `rugo_go_from_string_slice(goValue)` → `[]interface{}`
- `GoInt` from int64 returns (e.g. `time.Now().Unix()`): `interface{}(int(v))`

## Function Signature Registry

Each whitelisted function is described by `GoFuncSig`:

```go
type GoFuncSig struct {
    GoName   string    // PascalCase Go function name (e.g. "Contains")
    Params   []GoType  // Parameter types in order
    Returns  []GoType  // Return types in order
    Variadic bool      // Last param is variadic
}
```

### Return pattern handling

| Pattern | Codegen strategy |
|---------|-----------------|
| `()` void | Wrap in IIFE: `func() interface{} { pkg.Func(...); return nil }()` |
| `(T)` | `interface{}(pkg.Func(...))` |
| `(error)` | `func() interface{} { if err := ...; err != nil { panic(err.Error()) }; return nil }()` |
| `(T, error)` | IIFE with error check → panic on error, return T |
| `(T, bool)` | IIFE → return nil if false, return T if true |
| `(T1, T2)` tuple | Package-specific: `filepath.Split` returns as `[]interface{}{dir, file}` |

### Name conversion

Rugo `snake_case` → Go `PascalCase` is handled by the registry — each entry
explicitly maps the Rugo name to the Go name. There's no automatic conversion;
every function must be registered.

```go
"has_prefix": {GoName: "HasPrefix", Params: []GoType{GoString, GoString}, Returns: []GoType{GoBool}},
```

## Adding a New Go Package

### 1. Create a mapping file in `compiler/gobridge/`

Create `compiler/gobridge/newpkg.go` — it self-registers via `init()`:

```go
package gobridge

func init() {
    Register(&Package{
        Path: "newpkg",   // or "path/newpkg" for nested packages
        Funcs: map[string]GoFuncSig{
            "do_thing": {
                GoName:  "DoThing",
                Params:  []GoType{GoString, GoInt},
                Returns: []GoType{GoString, GoError},
            },
        },
    })
}
```

That's it — rebuild and the new package is available. No other files need editing.

**Versioned packages** (e.g. `math/rand/v2`): The `DefaultNS()` function
automatically strips Go version suffixes, so `math/rand/v2` uses namespace `rand`.
Users write `import "math/rand/v2"` and call `rand.int_n(10)`.

### 2. Handle special cases (if needed)

If the Go function has non-standard behavior, use the `Codegen` callback
on `GoFuncSig` — the bridge file owns its own codegen logic:

```go
"read_file": {
    GoName: "ReadFile", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
    Doc: "Reads and returns the contents of the named file.",
    Codegen: func(pkgBase string, args []string, rugoName string) string {
        return fmt.Sprintf("func() interface{} { _v, _err := %s.ReadFile(%s); if _err != nil { %s }; return interface{}(string(_v)) }()",
            pkgBase, TypeConvToGo(args[0], GoString), PanicOnErr(rugoName))
    },
},
```

The `Codegen` callback receives:
- `pkgBase` — resolved Go package name (respects `import "os" as go_os`)
- `args` — raw Go expressions for each argument
- `rugoName` — user-visible name for error messages (e.g. `"os.read_file"`)

When `Codegen` is nil, the generic handler in `codegen.go` handles the call
based on `Params`, `Returns`, and `Variadic` fields.

Special cases are needed for:
- **[]byte conversions**: `os.ReadFile` returns `[]byte` → string
- **Type casting**: `strconv.FormatInt` needs `int64()`, `os.MkdirAll` needs `os.FileMode()`
- **Method chains**: `time.Now().Unix()` — GoName contains `.`
- **Mutating functions**: `sort.Strings` — copy-in/copy-out pattern
- **Struct decomposition**: `url.Parse` → Rugo hash
- **Runtime-only packages**: `slices`, `maps` — no Go import, pure helper dispatch

### 2b. Declare runtime helpers (if needed)

If a bridge function needs Go helper functions emitted into the generated code,
declare them via `RuntimeHelpers`:

```go
var myHelpers = []RuntimeHelper{
    {Key: "rugo_my_helper", Code: `func rugo_my_helper(v interface{}) interface{} {
    // helper implementation
}

`},
}

"my_func": {
    GoName: "MyFunc", Params: []GoType{GoString}, Returns: []GoType{GoString},
    Codegen: func(pkgBase string, args []string, _ string) string {
        return fmt.Sprintf("rugo_my_helper(%s)", args[0])
    },
    RuntimeHelpers: myHelpers,
},
```

Helpers are deduplicated by `Key` — multiple functions sharing the same helper
only emit it once. Use the same `[]RuntimeHelper` slice for all functions that
share helpers.

### 2c. Package-level options

For packages implemented entirely via runtime helpers (no actual Go imports):

```go
Register(&Package{
    Path:         "slices",
    NoGoImport:   true,                  // don't emit `import "slices"` in generated Go
    ExtraImports: []string{"sort"},      // additional Go imports needed by helpers
    Funcs: map[string]GoFuncSig{ ... },
})
```

### 3. Add tests

Create `rats/gobridge/<pkg>_test.rugo`:

```ruby
use "test"

import "newpkg"

rats "newpkg.do_thing works"
  result = newpkg.do_thing("hello", 42)
  test.assert_eq(result, "expected")
end

rats "newpkg.do_thing error handling"
  result = try newpkg.do_thing("bad", -1) or "fallback"
  test.assert_eq(result, "fallback")
end
```

Run with:
```bash
go build -o bin/rugo . && go install .
bin/rugo rats rats/gobridge/
```

### 4. Verify with emit

Inspect the generated Go to confirm correct codegen:

```bash
echo 'import "newpkg"
puts newpkg.do_thing("hello", 42)' > /tmp/test.rugo
bin/rugo emit /tmp/test.rugo
```

## Special Case Catalog

These Go functions required custom codegen in `generateGoBridgeCall()`:

### time.Now().Unix() / time.Now().UnixNano()

Method-chain calls. `GoName` contains `.`, triggering the chain path.
Returns `int64`, wrapped with `int()` for Rugo:

```go
interface{}(int(time.Now().Unix()))
```

### time.Sleep

Accepts milliseconds in Rugo, converts to `time.Duration`. Void return
wrapped in IIFE:

```go
func() interface{} { time.Sleep(time.Duration(rugo_to_int(ms)) * time.Millisecond); return nil }()
```

### os.ReadFile

Go returns `[]byte` — bridge converts to `string`:

```go
func() interface{} { _v, _err := os.ReadFile(path); if _err != nil { panic(_err.Error()) }; return interface{}(string(_v)) }()
```

### os.MkdirAll

Second arg is `os.FileMode`, needs explicit cast:

```go
os.MkdirAll(path, os.FileMode(rugo_to_int(perm)))
```

### strconv.FormatInt / ParseInt

Go uses `int64`. Bridge casts between `int` and `int64`:

```go
// FormatInt
interface{}(strconv.FormatInt(int64(rugo_to_int(n)), rugo_to_int(base)))
// ParseInt
interface{}(int(_v))  // _v is int64 from ParseInt
```

### sort.Strings / sort.Ints

Mutate in-place. Bridge does copy-in/copy-out:

```go
func() interface{} {
    _s := rugo_go_to_string_slice(arr)
    sort.Strings(_s)
    return rugo_go_from_string_slice(_s)
}()
```

### filepath.Join

Variadic `...string`. Each Rugo arg converted individually (not as a slice):

```go
filepath.Join(rugo_to_string(a), rugo_to_string(b), rugo_to_string(c))
```

### filepath.Split

Returns `(dir, file)` tuple — mapped to Rugo array:

```go
func() interface{} {
    _d, _f := filepath.Split(path)
    return interface{}([]interface{}{interface{}(_d), interface{}(_f)})
}()
```

### encoding/json.Marshal / MarshalIndent

Go's `json.Marshal` can't handle Rugo's `map[interface{}]interface{}` type.
Bridge converts recursively via `rugo_json_prepare()` before marshaling,
and converts `[]byte` result to string:

```go
func() interface{} {
    _v, _err := json.Marshal(rugo_json_prepare(data))
    if _err != nil { panic(rugo_bridge_err("json.marshal", _err)) }
    return interface{}(string(_v))
}()
```

### encoding/json.Unmarshal

Go's pointer-based `json.Unmarshal` API wrapped in an IIFE. Result converted
from Go's `map[string]interface{}` back to Rugo's `map[interface{}]interface{}`
via `rugo_json_to_rugo()`:

```go
func() interface{} {
    var _v interface{}
    _err := json.Unmarshal([]byte(rugo_to_string(s)), &_v)
    if _err != nil { panic(rugo_bridge_err("json.unmarshal", _err)) }
    return rugo_json_to_rugo(_v)
}()
```

### encoding/base64 (encode/decode)

Method-chain calls on `StdEncoding`/`URLEncoding` package-level vars.
Encode takes `[]byte` input, decode returns `[]byte` → string:

```go
// encode
interface{}(base64.StdEncoding.EncodeToString([]byte(rugo_to_string(s))))
// decode
func() interface{} {
    _v, _err := base64.StdEncoding.DecodeString(rugo_to_string(s))
    if _err != nil { panic(rugo_bridge_err("base64.decode", _err)) }
    return interface{}(string(_v))
}()
```

### encoding/hex (encode/decode)

Same `[]byte` conversion pattern as base64:

```go
// encode
interface{}(hex.EncodeToString([]byte(rugo_to_string(s))))
// decode — returns []byte, convert to string
func() interface{} {
    _v, _err := hex.DecodeString(rugo_to_string(s))
    if _err != nil { panic(rugo_bridge_err("hex.decode", _err)) }
    return interface{}(string(_v))
}()
```

### crypto/sha256.Sum256 / crypto/md5.Sum

Go returns fixed-size arrays (`[32]byte`, `[16]byte`). Bridge converts to
hex string via `fmt.Sprintf`:

```go
func() interface{} {
    _h := sha256.Sum256([]byte(rugo_to_string(s)))
    return interface{}(fmt.Sprintf("%x", _h))
}()
```

### net/url.Parse — Struct decomposition

First bridge function that decomposes a Go struct into a Rugo hash.
`url.Parse` returns `*url.URL` — bridge extracts fields into a
`map[interface{}]interface{}` for dot-access:

```go
func() interface{} {
    _u, _err := url.Parse(rugo_to_string(s))
    if _err != nil { panic(rugo_bridge_err("url.parse", _err)) }
    _user := ""
    if _u.User != nil { _user = _u.User.Username() }
    return map[interface{}]interface{}{
        "scheme":   _u.Scheme,
        "host":     _u.Host,
        "hostname": _u.Hostname(),
        "port":     _u.Port(),
        "path":     _u.Path,
        "query":    _u.RawQuery,
        "fragment": _u.Fragment,
        "user":     _user,
        "raw":      _u.String(),
    }
}()
```

### unicode (is_letter, to_upper, etc.)

Go's `unicode` functions operate on `rune`. Bridge extracts the first rune
from the input string using `rugo_utf8_decode()`:

```go
func() interface{} {
    _s := rugo_to_string(arg)
    if len(_s) == 0 { return interface{}(false) }
    _r, _ := rugo_utf8_decode(_s)
    return interface{}(unicode.IsLetter(_r))
}()
```

### slices (contains, index, reverse, compact) — Runtime helpers

Go's `slices` package uses generics incompatible with `interface{}`.
Bridge emits custom runtime helpers using `fmt.Sprintf` for comparison:

```go
func rugo_slices_contains(v interface{}, target interface{}) interface{} {
    arr := v.([]interface{})
    ts := fmt.Sprintf("%v", target)
    for _, e := range arr {
        if fmt.Sprintf("%v", e) == ts { return interface{}(true) }
    }
    return interface{}(false)
}
```

### maps (keys, values, clone, equal) — Runtime helpers

Same approach as slices — runtime helpers on `map[interface{}]interface{}`.
Keys are sorted for deterministic output:

## Import Deduplication

The codegen tracks `emittedImports` to prevent duplicate Go import statements:

1. **Base imports** always emitted: `fmt`, `os`, `os/exec`, `runtime/debug`, `strings`
2. **Rugo module imports** (from `use`): e.g., `use "conv"` adds `strconv`
3. **Go bridge imports** (from `import`): added last, skipped if already emitted
4. **Aliased imports**: always emitted even if the bare path exists (Go allows `import alias "path"`)

Example conflict: `use "conv"` pulls in `strconv`, and `import "strconv"` also
wants it. The dedup map prevents the redeclaration error.

## Namespace Conflict Detection

The compiler rejects ambiguous namespaces at compile time:

| Conflict | Error |
|----------|-------|
| `use "os"` + `import "os"` | Must alias: `import "os" as go_os` |
| `require "x" as "os"` + `use "os"` | Require alias conflicts with use'd module |
| `require "x" as "strings"` + `import "strings"` | Require alias conflicts with Go bridge |

## Runtime Helpers

Bridge functions declare their runtime helpers via the `RuntimeHelpers` field
on `GoFuncSig`. The codegen scans all imported packages, collects helpers,
deduplicates by key, and emits them once. No hardcoded package name checks.

Currently declared helpers:

- `rugo_go_to_string_slice` / `rugo_go_from_string_slice` — declared by sort and strings functions using `GoStringSlice`
- `rugo_go_to_int_slice` / `rugo_go_from_int_slice` — declared by sort.Ints
- `rugo_utf8_decode` — declared by unicode functions (first rune extraction)
- `rugo_json_prepare` / `rugo_json_to_rugo` — declared by json marshal/unmarshal
- `rugo_slices_contains`, `rugo_slices_index`, `rugo_slices_reverse`, `rugo_slices_compact` — declared by slices functions
- `rugo_maps_keys`, `rugo_maps_values`, `rugo_maps_clone`, `rugo_maps_equal` — declared by maps functions

## Test Coverage

All 16 whitelisted packages have comprehensive regression tests in `rats/gobridge/`:

| File | Tests | Covers |
|------|-------|--------|
| `strings_test.rugo` | 14 | All string functions including cut, fields, equal_fold |
| `strconv_test.rugo` | 6 | atoi, itoa, format_float, parse_float, format_bool, parse_int |
| `math_test.rugo` | 9 | Arithmetic, trig, rounding, special values (NaN, Inf) |
| `filepath_test.rugo` | 8 | join, base, dir, ext, split, is_abs, rel, clean |
| `rand_test.rugo` | 5 | int_n, float64, n alias, range validation |
| `misc_test.rugo` | 10 | sort, time, os (env, files, dirs) |
| `edge_cases_test.rugo` | 8 | Error handling, aliasing, namespace conflicts, try/or |
| `json_test.rugo` | 12 | marshal, unmarshal, marshal_indent, round-trip, nested, nil, booleans, errors |
| `base64_test.rugo` | 6 | encode/decode, url_encode/url_decode, round-trip, errors, special chars |
| `hex_test.rugo` | 5 | encode/decode, known values, round-trip, errors |
| `crypto_test.rugo` | 7 | sha256/md5 known hashes, empty string, consistency, different inputs |
| `url_test.rugo` | 8 | parse (full/partial/relative), escape/unescape, aliases, errors |
| `unicode_test.rugo` | 8 | is_letter/digit/space/upper/lower/punct, to_upper/to_lower, empty, first char |
| `slices_test.rugo` | 8 | contains, index, reverse, compact, string/int types, empty, immutability |
| `maps_test.rugo` | 9 | keys, values, clone, equal, empty hash, shallow clone |

Run all: `rugo rats rats/gobridge/`
