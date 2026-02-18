# Go Bridge Internals

This document explains how Rugo's Go bridge works: how `import "pkg"` gets resolved, how function signatures are classified, how calls are generated, and how structs/methods are exposed safely to Rugo.

---

## What the bridge does

The Go bridge lets Rugo code call selected Go packages directly:

```rugo
import "strings"
puts strings.to_upper("hello")
```

At compile time, Rugo inspects Go package symbols, classifies bridgeable functions/types, registers metadata, and emits type-safe Go calls with runtime conversions between Rugo values (`interface{}`) and concrete Go types.

---

## High-level architecture

### Main components

- `gobridge/gobridge.go`
  - Core registry (`Package`, `GoFuncSig`, `GoType`, helpers)
  - Namespace resolution and type conversion helpers
- `gobridge/inspect.go`
  - Package introspection (`InspectCompiledPackage`, `InspectSourcePackage`)
  - Function/var/const/method classification
  - Struct discovery and wrapper reclassification
- `gobridge/classify.go`
  - Type classifier (`ClassifyGoType`, `ClassifyFunc`, tiers)
- `gobridge/struct_wrap.go`
  - Generated wrapper helpers for struct handles (`DotGet`, `DotSet`, `DotCall`)
- `compiler/codegen_expr.go`
  - Resolves `ns.func(...)` calls as Go bridge calls
- `compiler/codegen_runtime.go`
  - Emits final Go call expressions from bridge metadata

---

## Two bridge entry points

### 1) `import "go/pkg"` (compiled package introspection)

When the compiler sees `ImportStmt`, it ensures the package is in the bridge registry:

1. `compiler/compiler.go` calls `gobridge.InspectCompiledPackage(...)` on first use.
2. Inspector reads exported symbols from type info (`go/importer`).
3. Classifier maps bridgeable functions and constants/vars into `GoFuncSig`.
4. Registry stores package metadata under full import path.

This path is used for stdlib bridge usage (e.g. `time`, `strings`, `os`, `net/url`).

### 2) `require "<go module dir>"` (source package introspection)

For Go modules loaded through `require`, the pipeline is richer:

1. `InspectSourcePackage` parses/type-checks source files.
2. `FinalizeStructs` discovers struct methods and external dependency types.
3. Blocked functions are reclassified if they become bridgeable via generated wrappers.
4. The package is registered as `External`, and codegen emits import/use wiring.

This is how user Go modules gain struct handle support and method dispatch in Rugo.

---

## Registry model

`Package` is the unit of registration:

- `Path`: full Go import path
- `Funcs`: map of `rugo_name -> GoFuncSig`
- `Structs`: discovered struct metadata (for wrapper generation)
- `ExtraImports`: additional Go imports needed by helper code
- `External`: true for `require`d Go modules

Bridge lookups happen by:

- package path (`Lookup(pkg, rugoName)`)
- namespace (`PackageForNS`) using explicit alias first, then default namespace.

---

## Signature classification

### Type tiers

Each function is classified into one of:

- `TierAuto`: directly bridgeable
- `TierCastable`: bridgeable with explicit cast/conversion
- `TierFunc`: has function-typed parameters (lambda adapter needed)
- `TierBlocked`: unsupported shape (generics, maps/chans/interfaces with methods, etc.)

### GoType mapping

The bridge normalizes Go signatures into `GoType` enums (e.g. `GoString`, `GoInt`, `GoByteSlice`, `GoFunc`, `GoError`, `GoAny`).

Notable behavior:

- named aliases/basic wrappers (e.g. `os.FileMode`) carry explicit `TypeCasts`
- `(T, error)` and `(T, bool)` returns get special runtime behavior
- fixed-size array returns can be represented through `ArrayTypes` metadata
- byte-array *parameters* are intentionally blocked unless explicitly handled

### Vars and consts

Exported Go vars/consts are exposed as zero-arg bridge accessors:

- `math.Pi` -> `math.pi()`
- `time.RFC3339` -> `time.rfc3339()`

---

## Struct bridging model

Structs are exposed as opaque handles with dot support:

- `DotGet(field)` for reads
- `DotSet(field, val)` for writes (typed conversion + optional cast)
- `DotCall(method, args...)` for method dispatch

Generated wrapper type names are deterministic:

- in-package: `rugo_struct_<ns>_<GoType>`
- external deps: `rugo_ext_<ns>_<pkg>_<GoType>`

### Reclassification

Initially blocked functions are retried with wrapper-aware logic:

- struct params map to wrapper unwrapping (`StructCasts`)
- struct returns map to wrapper wrapping (`StructReturnWraps`)
- value-vs-pointer metadata (`StructParamValue`, `StructReturnValue`) controls dereference/address semantics

### Upcasting support

For embedded struct hierarchies, upcast helpers are generated (`rugo_upcast_<wrapper>`), so derived wrappers can satisfy base-type params.

---

## Lambda adapter support (`GoFunc`)

If a Go API expects a function parameter:

1. Signature is captured as `GoFuncType`.
2. Codegen emits adapter closure converting between typed Go params and Rugo variadic lambda call convention.
3. Struct callback params are wrapped/unwrapped when needed.

This enables APIs like predicates, mappers, and callback-style methods.

---

## Code generation path

When codegen sees `ns.func(...)`:

1. `codegen_expr.go` resolves namespace to package and function signature.
2. `generateGoBridgeCall(...)` builds Go call expression from `GoFuncSig`:
   - argument conversion (`TypeConvToGo`)
   - named type casts
   - struct unwrap/wrap
   - variadic handling
   - return pattern lowering

### Return pattern rules

- `()` -> returns `nil`
- `(error)` -> panic on non-nil
- `(T, error)` -> panic on error, return `T`
- `(T, bool)` -> return `nil` when bool is false
- multi-return -> `[]interface{}{...}`

Panics are formatted through `rugo_bridge_err(...)`, so `try/or` can catch bridge failures predictably.

---

## Runtime helpers

Bridge functions can request helper snippets (`RuntimeHelpers`) that are emitted once per program:

- rune extraction
- string-slice conversion
- byte-slice conversion
- struct wrapper types and upcast helpers
- package-specific helper code for custom codegen paths

Codegen deduplicates helpers by key and appends required `ExtraImports`.

---

## Current stdlib package set

As of current registry defaults (`stdlibPackages`):

- `crypto/md5`
- `crypto/sha256`
- `encoding/base64`
- `encoding/hex`
- `encoding/json`
- `html`
- `math`
- `math/rand/v2`
- `net/url`
- `os`
- `path`
- `path/filepath`
- `sort`
- `strconv`
- `strings`
- `time`
- `unicode`

---

## Error semantics and validation

- Unknown bridge function calls are compile-time errors.
- Identifier checker validates namespaced calls against bridge registry.
- Go runtime errors from bridge calls surface as Rugo bridge errors and integrate with `try/or`.

---

## Extending the bridge

### Add support for new signature shapes

Start in:

- `ClassifyGoType` / `ClassifyFunc` (`gobridge/classify.go`)
- reclassification/wrapper logic (`gobridge/inspect.go`)
- conversions (`TypeConvToGo`, `TypeWrapReturn` in `gobridge/gobridge.go`)

### Add package-specific behavior

Use `GoFuncSig.Codegen` callback for custom call emission when generic rules are not enough.

### Add runtime helper logic

Attach helper snippets via `GoFuncSig.RuntimeHelpers`; ensure imports are reflected via `ExtraImports` where needed.

---

## Debugging workflow

Useful commands:

```bash
go run . emit script.rugo              # inspect generated Go
go test ./gobridge ./compiler -count=1
bin/rugo rats --recap --timing rats/gobridge/
```

For bridge issues, inspect generated code first (`emit`) to verify:

- selected Go package alias
- argument conversion and casts
- wrapper generation/usage
- return lowering pattern
