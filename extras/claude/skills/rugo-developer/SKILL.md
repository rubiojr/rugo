---
name: rugo-developer
description: Expert in developing Rugo, a Ruby-inspired language that compiles to native binaries via Go. Load when working on the rugo compiler, parser, modules, or writing .rg scripts.
---

# Rugo Developer Skill

## CRITICAL

* Rugo is written in Go.
* Avoid introducing insecure code; ask first. Security is paramount.
* Read `preference.md` and `rats.md` in the repo root for current design decisions and pending work before making changes.
* **Load the `rugo-quickstart` skill** when writing `.rg` scripts or helping users with Rugo language syntax and features.

## Project Overview

Rugo is a tiny Ruby-inspired language that compiles to native binaries via Go. The compiler pipeline is:

```
.rg source → preprocess → parse (EBNF grammar) → AST walk → Go codegen → go build
```

Repository: `github.com/rubiojr/rugo`

### Key directories

| Path | Purpose |
|------|---------|
| `main.go` | CLI entry point (urfave/cli): `run`, `build`, `emit`, `rats`, `bench`, `dev` subcommands |
| `parser/` | Generated LL(1) parser from `rugo.ebnf` (do NOT hand-edit `parser.go`) |
| `parser/rugo.ebnf` | Authoritative grammar — edit this to change syntax |
| `compiler/` | Compiler pipeline: preprocess → parse → walk → codegen |
| `compiler/preprocess.go` | Line-by-line preprocessor (shell fallback, paren-free calls, string interpolation) |
| `compiler/walker.go` | AST walker — transforms parse tree into compiler nodes |
| `compiler/codegen.go` | Go code generation from compiler nodes |
| `compiler/nodes.go` | AST node types |
| `compiler/gobridge/` | Go stdlib bridge: type registry and per-package mapping files |
| `compiler/compiler.go` | Orchestrates: file loading, require resolution, compilation |
| `cmd/dev/` | Developer tools: `modgen` module scaffolding |
| `modules/` | Stdlib module registry and built-in modules |
| `modules/module.go` | Module type, registry API, `CleanRuntime` helper |
| `modules/{os,http,conv,str,test,bench,fmt,re}/` | Built-in modules (each has registration + `runtime.go`) |
| `rats/` | RATS regression tests (`_test.rg` files) |
| `bench/` | Performance benchmarks (`_bench.rg` files) |
| `examples/` | Example `.rg` scripts |
| `docs/mods.md` | Module system documentation |
| `changes.md` | Syntax improvement proposals and status |
| `script/test` | Rugo test runner script (builds rugo, runs Go tests, discovers and runs examples) |

## Language Features

* Ruby-like syntax: `def/end`, `if/elsif/else/end`, `while/end`, `for/in/end`
* Compiles to native binaries — no runtime needed
* Shell fallback — unknown identifiers at top level run as shell commands
* Paren-free calls — `puts "hello"` (preprocessor rewrites to `puts("hello")`)
* String interpolation — `"Hello, #{name}!"`
* Rugo stdlib modules — `use "http"` → `http.get(url)`
* Go stdlib bridge — `import "strings"` → `strings.to_upper("hello")` (direct Go calls)
* User modules — `require "lib"` → `lib.func()`
* Arrays, hashes, closureless functions
* Global builtins: `puts`, `print`, `len`, `append`
* `for..in` loops — `for x in arr`, `for k, v in hash` (iterates arrays and hashes)
* `break` / `next` — loop control (compiles to Go `break`/`continue`)
* Index assignment — `arr[0] = x`, `hash["key"] = y`
* Compound assignment — `x += 1`, `x -= 1`, `*=`, `/=`, `%=` (preprocessor sugar)
* Error handling — `try expr or default`, `try expr or err ... end`
* Concurrency — `spawn` (goroutine + task handle), `parallel` (fan-out N, wait all)
* Task API — `task.value` (block+get result), `task.done` (non-blocking check), `task.wait(n)` (timeout)

## Compilation Pipeline

### 1. Preprocessor (`compiler/preprocess.go`)

Transforms raw `.rg` source before parsing:
- Desugars compound assignment: `x += y` → `x = x + y` (also works with index targets like `arr[0] += 1`)
- Rewrites paren-free function calls: `puts "hi"` → `puts("hi")`
- Expands string interpolation: `"Hello, #{name}"` → `("Hello, " + __to_s(name) + "")`
- Converts unknown top-level identifiers to shell fallback: `ls -la` → `__shell__("ls -la")`
- Expands single-line `try` sugar into block form (skips block keywords like `spawn`, `parallel`)
- Expands single-line `spawn EXPR` into `spawn\n  EXPR\nend`
- Handles `use`/`import`/`require` statements
- Tracks block nesting via `blockStack` for `spawn`/`parallel` end-matching

**Keywords** (not treated as shell commands): `if`, `elsif`, `else`, `end`, `while`, `for`, `in`, `def`, `return`, `require`, `break`, `next`, `true`, `false`, `nil`, `use`, `import`, `test`, `try`, `or`, `spawn`, `parallel`, `bench`

**Important:** Shell fallback resolution is positional at top level (function names are only recognized after their `def` line) but forward-referencing inside function bodies. See `preference.md` for details.

### 2. Parser (`parser/`)

- Generated from `parser/rugo.ebnf` using the `egg` tool
- **Do NOT hand-edit `parser.go`** — regenerate from the EBNF
- To regenerate: `egg -o parser.go -package parser -start Program -type Parser -constprefix Rugo rugo.ebnf`

### 3. AST Walker (`compiler/walker.go`)

Walks the parse tree and produces typed AST nodes defined in `compiler/nodes.go`.

### 4. Code Generation (`compiler/codegen.go`)

Converts AST nodes to Go source code. Emits:
- A `main()` function with top-level code
- User-defined functions as Go functions
- Module runtime code and wrapper functions
- Shell fallback via `exec.Command("sh", "-c", ...)`
- `for..in` loops via `rugo_iterable()` (returns `[]rugo_kv` for uniform array/hash iteration)
- Index assignment via `rugo_index_set()` (type-switches arrays and hashes)
- `break`/`next` as Go `break`/`continue`
- `spawn` as IIFE with goroutine + `rugoTask` struct (result, error capture, done channel)
- `parallel` as IIFE with `sync.WaitGroup` + `sync.Once` (indexed goroutines, ordered results)
- Task method dispatch (`.value`/`.done`/`.wait`) via runtime helpers with friendly error messages
- Import gating: `sync`+`time` conditionally emitted based on `hasSpawn`/`hasParallel`/`usesTaskMethods` AST flags

## Module System

Rugo has three import mechanisms:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules (hand-crafted wrappers) | `use "http"` |
| `import` | Go stdlib bridge (auto-generated calls) | `import "strings"` |
| `require` | User `.rg` files | `require "helpers"` |

### Rugo Stdlib Modules (`use`)

Modules self-register via Go `init()` using `modules.Register()`. Each module has:
- `runtime.go` — Go struct + methods (tagged `//go:build ignore`, embedded as string)
- Registration file — embeds `runtime.go`, declares function signatures with typed args

### Creating a new module

Use the module generator to scaffold:

```bash
rugo dev modgen mymod --funcs do_thing,other_func
```

This creates `modules/mymod/` with registration, runtime, and stubs files, and adds the blank import to `main.go`. Fill in the method implementations in `runtime.go`.

Manual steps:
1. Create `modules/mymod/` with `mymod.go` (registration) and `runtime.go` (implementation)
2. Runtime methods use typed parameters on a struct receiver (not `interface{}`)
3. Register with `modules.Register()` specifying `Name`, `Type`, `Funcs`, `GoImports`, `Runtime`
4. Add blank import in `main.go`: `_ "github.com/rubiojr/rugo/modules/mymod"`
5. See `docs/mods.md` for the full reference

### Available argument types for `FuncDef.Args`

| ArgType | Go type | Conversion function |
|---------|---------|---------------------|
| `String` | `string` | `rugo_to_string` |
| `Int` | `int` | `rugo_to_int` |
| `Float` | `float64` | `rugo_to_float` |
| `Bool` | `bool` | `rugo_to_bool` |
| `Any` | `interface{}` | none |

## Go Stdlib Bridge (`import`)

The `import` keyword gives direct access to whitelisted Go stdlib packages. The compiler generates type-safe Go calls with `interface{}` ↔ typed conversions.

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")   # HELLO
puts math.sqrt(144.0)            # 12
```

Function names use `snake_case` in Rugo, auto-mapped to Go's `PascalCase` via the registry.

### Bridge architecture

- **Registry**: `compiler/gobridge/gobridge.go` — types (`GoType`, `GoFuncSig`, `Package`), registry API (`Register`, `IsPackage`, `Lookup`, `PackageNames`)
- **Mapping files**: `compiler/gobridge/{strings,strconv,math,rand,filepath,sort,os,time}.go` — each self-registers via `init()`
- **Codegen**: `compiler/codegen.go` `generateGoBridgeCall()` — emits direct Go calls with type conversions and special-case handling
- **Adding a new bridge**: Create `compiler/gobridge/newpkg.go` with `init()` calling `Register()` — no other files need editing

### Whitelisted packages

`strings`, `strconv`, `math`, `math/rand/v2`, `path/filepath`, `sort`, `os`, `time`

### Key special cases in codegen

- `time.Sleep` — ms→Duration conversion, void wrapped in IIFE
- `time.Now().Unix()` — method-chain calls, int64→int cast
- `os.ReadFile` — []byte→string conversion
- `os.MkdirAll` — os.FileMode cast for permissions
- `strconv.FormatInt`/`ParseInt` — int64 conversions
- `sort.Strings`/`Ints` — copy-in/copy-out (mutate-in-place bridge)
- `filepath.Split` — tuple return mapped to Rugo array
- `filepath.Join` — variadic string args

### Error handling

Go `(T, error)` returns auto-panic, integrating with `try/or`:
```ruby
n = try strconv.atoi("abc") or 0
```

### Aliasing

`import "os" as go_os` when namespace conflicts with `use "os"`.

See `docs/gobridge.md` for the full developer reference.

## Building & Testing

### Build

```bash
go build -o bin/rugo .
```

### Run Go tests

```bash
go test ./... -count=1
```

### RATS Regression Tests

RATS (Rugo Automated Testing System) tests live in `rats/` as `_test.rg` files. **Load the `rugo-rats` skill** for full details on test syntax, assertions, the test runner, and the regression test suite.

```bash
rugo rats rats/                    # run all _test.rg files in rats/
rugo rats rats/03_control_flow_test.rg  # run a specific test file
```

### Benchmarks

Benchmarks use the `bench` keyword and `_bench.rg` file convention (like Go's `_test.go`):

```bash
rugo bench bench/                        # run all _bench.rg files in bench/
rugo bench bench/arithmetic_bench.rg     # run a specific benchmark
rugo bench bench/io_bench.rg 1>/dev/null # redirect program stdout, keep bench output on stderr
```

Benchmark files use `use "bench"` and `bench` blocks:

```ruby
use "bench"

bench "fib(20)"
  fib(20)
end
```

The framework auto-calibrates iterations (scales until ≥1s elapsed), reports ns/op and run count. Output goes to stderr with ANSI colors (respects `NO_COLOR`).

Benchmark files in `bench/`: `arithmetic_bench.rg`, `strings_bench.rg`, `collections_bench.rg`, `control_flow_bench.rg`, `io_bench.rg`.

### Go-side Compiler Benchmarks

```bash
go test -bench=. ./compiler/ -benchmem
```

Covers: `CompileHelloWorld`, `CompileFunctions`, `CompileControlFlow`, `CompileStringInterpolation`, `CompileArraysHashes`, `Preprocess`, `Codegen`.

### Run the full test suite (Go tests + all examples)

```bash
rugo run script/test
```

### Test a .rg script

```bash
go run . run examples/hello.rg
go run . emit examples/hello.rg   # inspect generated Go code
```

### Inspect generated Go for debugging

Use `emit` to see what Go code is produced — this is the best way to debug codegen issues:

```bash
go run . emit script.rg
```

## Development Workflow

1. **Before editing**, read relevant source files and understand the pipeline stage you're modifying.
2. **Grammar changes**: Edit `rugo.ebnf`, regenerate `parser.go`, then update the walker and codegen.
3. **Preprocessor changes**: Be careful with shell fallback logic — read `preference.md` for the positional resolution design.
4. **New modules**: Follow the pattern in `docs/mods.md`. Always add typed function signatures.
5. **After edits**: Run `go test ./... -count=1` and test with relevant examples.
6. **Format code**: `go fmt ./...`

## Concurrency

Rugo has two concurrency primitives backed by goroutines. See `docs/concurrency.md` for the full design doc.

### `spawn` — single goroutine + task handle

```ruby
# Block form
task = spawn
  http.get("https://example.com")
end

# One-liner sugar (preprocessor expands to block form)
task = spawn http.get("https://example.com")

# Fire-and-forget (no assignment)
spawn
  puts "background work"
end

# Task API
task.value      # block until done, return result (re-raises panics)
task.done       # non-blocking: true if finished
task.wait(5)    # block with timeout, panics on timeout
```

**Codegen:** IIFE creating `rugoTask{done: make(chan struct{})}`, goroutine with defer/recover that captures panics into `t.err` and closes `t.done`. Last expression → `t.result`.

### `parallel` — fan-out N expressions, wait for all

```ruby
results = parallel
  http.get("https://api1.com")
  http.get("https://api2.com")
end
puts results[0]
```

**Codegen:** IIFE with `sync.WaitGroup` + `sync.Once` for first-error capture. Each statement gets its own goroutine writing to `_results[i]`. Returns `[]interface{}`.

### Key implementation details

- **Task method dispatch is always-on** — `.value`/`.done`/`.wait` always compile to `rugo_task_*` helpers, not gated on `hasSpawn`. The `usesTaskMethods` AST scan independently gates runtime emission.
- **Import gating:** `hasSpawn` → `sync`+`time`; `hasParallel` → `sync` only; `usesTaskMethods` → `sync`+`time`
- **Error messages:** Runtime helpers type-check and produce friendly messages: `cannot call .value on int — expected a spawn task`
- **Empty body:** `spawn` returns nil task; `parallel` returns `[]interface{}{}`
- **Try sugar interaction:** `expandTrySugar` skips lines where expression starts with block keyword (`spawn`/`parallel`)
- **One-liner limitation:** `spawn EXPR` works at line-start or after `=`, not nested in function calls

### RATS tests

See the `rugo-rats` skill for the full regression test inventory. Key test files:

- `rats/13_spawn_test.rg`, `rats/14_parallel_test.rg`, `rats/28_bench_test.rg`
- `rats/gobridge/` — 60 tests across 7 files covering all 8 Go bridge packages
- Fixtures in `rats/fixtures/`

## Common Pitfalls

* `parser.go` is generated — never edit it directly; edit `rugo.ebnf` and regenerate.
* Shell fallback is the default for unknown identifiers at top level — new builtins/keywords must be added to the preprocessor's known sets to avoid being treated as shell commands.
* Module `runtime.go` files have `//go:build ignore` tags — they're embedded as strings, not compiled directly.
* The preprocessor runs before parsing — some syntax transformations happen there, not in the parser/walker.
