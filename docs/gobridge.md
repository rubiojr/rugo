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

If the Go function has non-standard behavior, add a case in
`generateGoBridgeCall()` in `compiler/codegen.go`:

```go
// Handle newpkg.DoThing special case
if pkg == "newpkg" && sig.GoName == "DoThing" {
    return fmt.Sprintf("/* custom codegen */")
}
```

Special cases are needed for:
- **Method chains**: `time.Now().Unix()` — GoName contains `.`
- **int64 ↔ int**: Go returns `int64`, Rugo only knows `int`
- **Mutating functions**: `sort.Strings` mutates a slice — needs copy-in/copy-out
- **Variadic string args**: `filepath.Join("a", "b", "c")` — each arg converted individually
- **Tuple returns**: `filepath.Split` — mapped to Rugo array
- **FileMode/Duration**: `os.MkdirAll` perm, `time.Sleep` milliseconds
- **[]byte returns**: `os.ReadFile` — converted to string

### 3. Add tests

Create `rats/gobridge/<pkg>_test.rg`:

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
puts newpkg.do_thing("hello", 42)' > /tmp/test.rg
bin/rugo emit /tmp/test.rg
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

When Go bridge calls use `GoStringSlice` or sort, the codegen emits helper
functions. These are only emitted when needed (checked by scanning the
imported packages):

- `rugo_go_to_string_slice(v interface{}) []string` — converts `[]interface{}` to `[]string`
- `rugo_go_from_string_slice(v []string) interface{}` — converts `[]string` to `[]interface{}`
- `rugo_go_to_int_slice(v interface{}) []int` — converts `[]interface{}` to `[]int`
- `rugo_go_from_int_slice(v []int) interface{}` — converts `[]int` to `[]interface{}`

## Test Coverage

All 8 whitelisted packages have comprehensive regression tests in `rats/gobridge/`:

| File | Tests | Covers |
|------|-------|--------|
| `strings_test.rg` | 14 | All string functions including cut, fields, equal_fold |
| `strconv_test.rg` | 6 | atoi, itoa, format_float, parse_float, format_bool, parse_int |
| `math_test.rg` | 9 | Arithmetic, trig, rounding, special values (NaN, Inf) |
| `filepath_test.rg` | 8 | join, base, dir, ext, split, is_abs, rel, clean |
| `rand_test.rg` | 5 | int_n, float64, n alias, range validation |
| `misc_test.rg` | 10 | sort, time, os (env, files, dirs) |
| `edge_cases_test.rg` | 8 | Error handling, aliasing, namespace conflicts, try/or |

Run all: `rugo rats rats/gobridge/`
