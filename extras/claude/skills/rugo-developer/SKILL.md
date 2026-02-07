---
name: rugo-developer
description: Expert in developing Rugo, a Ruby-inspired language that compiles to native binaries via Go. Load when working on the rugo compiler, parser, modules, or writing .rg scripts.
---

# Rugo Developer Skill

## CRITICAL

* Rugo is written in Go.
* Avoid introducing insecure code; ask first. Security is paramount.
* Read `preference.md` and `rats.md` in the repo root for current design decisions and pending work before making changes.

## Project Overview

Rugo is a tiny Ruby-inspired language that compiles to native binaries via Go. The compiler pipeline is:

```
.rg source → preprocess → parse (EBNF grammar) → AST walk → Go codegen → go build
```

Repository: `github.com/rubiojr/rugo`

### Key directories

| Path | Purpose |
|------|---------|
| `main.go` | CLI entry point (urfave/cli): `run`, `build`, `emit` subcommands |
| `parser/` | Generated LL(1) parser from `rugo.ebnf` (do NOT hand-edit `parser.go`) |
| `parser/rugo.ebnf` | Authoritative grammar — edit this to change syntax |
| `compiler/` | Compiler pipeline: preprocess → parse → walk → codegen |
| `compiler/preprocess.go` | Line-by-line preprocessor (shell fallback, paren-free calls, string interpolation) |
| `compiler/walker.go` | AST walker — transforms parse tree into compiler nodes |
| `compiler/codegen.go` | Go code generation from compiler nodes |
| `compiler/nodes.go` | AST node types |
| `compiler/compiler.go` | Orchestrates: file loading, require resolution, compilation |
| `modules/` | Stdlib module registry and built-in modules |
| `modules/module.go` | Module type, registry API, `CleanRuntime` helper |
| `modules/{os,http,conv,str,test}/` | Built-in modules (each has registration + `runtime.go`) |
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
* Modules with namespaces — `import "http"` → `http.get(url)`
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
- Handles `import`/`require` statements
- Tracks block nesting via `blockStack` for `spawn`/`parallel` end-matching

**Keywords** (not treated as shell commands): `if`, `elsif`, `else`, `end`, `while`, `for`, `in`, `def`, `return`, `require`, `break`, `next`, `true`, `false`, `nil`, `import`, `test`, `try`, `or`, `spawn`, `parallel`

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

Modules self-register via Go `init()` using `modules.Register()`. Each module has:
- `runtime.go` — Go struct + methods (tagged `//go:build ignore`, embedded as string)
- Registration file — embeds `runtime.go`, declares function signatures with typed args

### Creating a new module

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

## Building & Testing

### Build

```bash
go build -o rugo .
```

### Run tests

```bash
go test ./... -count=1
```

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

- `rats/13_spawn.rt` — 21 tests (block, one-liner, fan-out, try/or, .done, .wait, functions, empty body, codegen gating, native binary, 5 negative tests)
- `rats/14_parallel.rt` — 11 tests (ordered results, shell commands, single expr, nested, try/or, empty body, import gating, native binary, 2 negative tests)
- Fixtures in `rats/fixtures/spawn_*.rg`, `rats/fixtures/err_spawn_*.rg`, `rats/fixtures/parallel_*.rg`, `rats/fixtures/err_parallel_*.rg`

## Common Pitfalls

* `parser.go` is generated — never edit it directly; edit `rugo.ebnf` and regenerate.
* Shell fallback is the default for unknown identifiers at top level — new builtins/keywords must be added to the preprocessor's known sets to avoid being treated as shell commands.
* Module `runtime.go` files have `//go:build ignore` tags — they're embedded as strings, not compiled directly.
* The preprocessor runs before parsing — some syntax transformations happen there, not in the parser/walker.
