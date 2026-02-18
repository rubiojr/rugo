---
name: rugo-developer
description: Expert in developing Rugo, a Ruby-inspired language that compiles to native binaries via Go. Load when working on the rugo compiler, parser, modules, or writing .rugo scripts.
---

# Rugo Developer Skill

## CRITICAL

* Rugo is written in Go.
* Avoid introducing insecure code; ask first. Security is paramount.
* Read `preference.md` and `rats.md` in the repo root for current design decisions and pending work before making changes.
* **Load the `rugo-quickstart` skill** when writing `.rugo` scripts or helping users with Rugo language syntax and features.
* **Load the `rugo-native-module-writer`** when you are writing or debugging a Rugo module written in Rugo lang.
* **Load the `idiomatic-rugo`** skill when writing Rugo code.

## Project Overview

Rugo is a tiny Ruby-inspired language that compiles to native binaries via Go. The compiler pipeline is:

```
.rugo source → strip comments → preprocess → parse (EBNF grammar) → AST walk → resolve imports/requires → Go codegen → go build
```

Repository: `github.com/rubiojr/rugo`

### Key directories

| Path | Purpose |
|------|---------|
| `main.go` | CLI entry point (urfave/cli): `run`, `build`, `emit`, `rats`, `bench`, `doc`, `mod`, `tool`, `dev` subcommands |
| `parser/` | Generated LL(1) parser from `rugo.ebnf` (do NOT hand-edit `parser.go`) |
| `parser/rugo.ebnf` | Authoritative grammar — edit this to change syntax |
| `ast/` | AST package: preprocessor, walker, and typed node definitions |
| `ast/preprocess.go` | Multi-pass preprocessor (compound assignment, bare append, backticks, try sugar, pipes, paren-free calls, shell fallback) |
| `ast/walker.go` | AST walker — transforms parse tree into typed AST nodes |
| `ast/nodes.go` | Typed AST node definitions (Statement and Expr interfaces) |
| `ast/cache.go` | Build cache management |
| `compiler/` | Compiler pipeline: codegen + orchestration |
| `compiler/codegen.go` | Go code generation from AST nodes |
| `compiler/compiler.go` | Orchestrates: file loading, require resolution, compilation |
| `remote/` | Remote module resolution, lockfile management |
| `cmd/` | CLI command implementations |
| `cmd/dev/` | Developer tools: `modgen` module scaffolding |
| `doc/` | Documentation extractor: parses `.rugo` doc comments, formats module/bridge docs |
| `modules/` | Stdlib module registry and built-in modules |
| `modules/module.go` | Module type, registry API, `CleanRuntime` helper |
| `modules/{ast,bench,cli,color,conv,fmt,http,json,os,queue,re,sqlite,str,test,web}/` | Built-in modules (each has registration + `runtime.go`) |
| `rats/` | RATS regression tests organized into subdirectories |
| `rats/core/` | Core language tests (variables, control flow, functions, lambdas, structs, pipes, etc.) |
| `rats/stdlib/` | Stdlib module tests (cli, color, http, json, web, queue, sqlite, etc.) |
| `rats/tools/` | Tool tests (linter, tool command, runmd) |
| `rats/fixtures/` | Test fixture files |
| `bench/` | Performance benchmarks (`_bench.rugo` files) |
| `examples/` | Example `.rugo` scripts |
| `tools/` | Rugo CLI extensions (linter, fuzz, runmd) |
| `docs/` | Full documentation (language.md, modules.md, etc.) |
| `script/test` | Rugo test runner script |

## Language Features

* Ruby-like syntax: `def/end`, `if/elsif/else/end`, `while/end`, `for/in/end`
* Compiles to native binaries — no runtime needed
* Shell fallback — unknown identifiers at top level run as shell commands
* Paren-free calls — `puts "hello"` (preprocessor rewrites to `puts("hello")`)
* String interpolation — `"Hello, #{name}!"`
* Raw strings — `'no #{interpolation}\n'` (single-quoted, no escape processing)
* Constants — uppercase identifiers (`PI = 3.14`) are immutable bindings (compile-time error on reassignment)
* Lambdas — `fn(params) body end` (first-class functions with closures)
* Structs — `struct Name fields end` with methods via `def Name.method(self, ...)`
* Pipe operator — `echo "hello" | str.upper | puts` (chains shell + Rugo functions)
* Built-in collection methods — `.map()`, `.filter()`, `.reduce()`, `.each()`, `.sort_by()`, `.join()`, etc. on arrays and hashes
* Rugo stdlib modules — `use "http"` → `http.get(url)`
* Go stdlib bridge — `import "strings"` → `strings.to_upper("hello")` (direct Go calls, 16 packages)
* User modules — `require "lib"` → `lib.func()`
* Remote modules — `require "github.com/user/repo@v1.0.0"` with lock files
* Selective imports — `require "lib" with client, helpers`
* Arrays, hashes, functions, lambdas
* Global builtins: `puts`, `print`, `len`, `append`, `raise`, `type_of`, `exit`
* Bare append sugar — `append arr, val` (preprocessor desugars to `arr = append(arr, val)`)
* `for..in` loops — `for x in arr`, `for k, v in hash` (iterates arrays and hashes)
* `break` / `next` — loop control (compiles to Go `break`/`continue`)
* Index assignment — `arr[0] = x`, `hash["key"] = y`
* Negative indexing — `arr[-1]` returns last element (Ruby behavior)
* Slicing — `arr[start, length]`, `str[start, length]` (clamped, Ruby behavior)
* Compound assignment — `x += 1`, `x -= 1`, `*=`, `/=`, `%=` (preprocessor sugar)
* Error handling — `try expr`, `try expr or default`, `try expr or err ... end`
* Concurrency — `spawn` (goroutine + task handle), `parallel` (fan-out N, wait all)
* Task API — `task.value` (block+get result), `task.done` (non-blocking check), `task.wait(n)` (timeout)
* Doc comments — `#` blocks before `def`/`struct` shown by `rugo doc`
* CLI extensions — `rugo tool install/list/remove` for compiled Rugo tools

### Variable Scoping

| Block | Own scope? | Sees outer vars? | Vars leak out? |
|-------|-----------|-------------------|----------------|
| **Top-level** | Yes (root) | — | — |
| **`def` function** | Yes | Yes (read-only) | No |
| **`fn` lambda** | Yes | Yes (captures outer) | No |
| **`if/elsif/else`** | No (transparent) | Yes | Yes |
| **`while` loop** | Yes | Yes (read + modify) | No |
| **`for..in` loop** | Yes | Yes (read + modify) | No |
| **`spawn` block** | Yes | Yes (shared) | No |
| **`rats` block** | Yes | No (isolated) | No |

## Compilation Pipeline

### 1. Preprocessor (`ast/preprocess.go`)

Transforms raw `.rugo` source before parsing. Operates in multiple passes:

**Pass 1: Compound assignment** — `x += y` → `x = x + y` (also index targets: `arr[0] += 1`)
**Pass 1b: Bare append** — `append arr, val` → `arr = append(arr, val)`
**Pass 2: Backtick expansion** — `` `hostname` `` → `__capture__("hostname")`
**Pass 3: Try sugar** — `try EXPR or DEFAULT` → multi-line block form
**Pass 4: Line-by-line processing:**
1. Pipe expansion — `echo "hi" | str.upper | puts` → nested calls (all-shell pipes left native)
2. Keywords — left untouched
3. Assignments — left untouched
4. Parenthesized calls — left untouched
5. Known function, paren-free — `puts "hi"` → `puts("hi")`
6. identifier Unknown `ls -la` → `__shell__("ls -la")` 

**Keywords** (not treated as shell commands): `if`, `elsif`, `else`, `end`, `while`, `for`, `in`, `def`, `return`, `require`, `break`, `next`, `true`, `false`, `nil`, `use`, `import`, `test`, `try`, `or`, `spawn`, `parallel`, `bench`, `struct`, `fn`

**Important:** Shell fallback resolution is positional at top level (function names are only recognized after their `def` line) but forward-referencing inside function bodies. See `preference.md` for details.

### 2. Parser (`parser/`)

- Generated from `parser/rugo.ebnf` using the `egg` tool
- **Do NOT hand-edit `parser.go`** — regenerate from the EBNF
- To regenerate: `egg -o parser.go -package parser -start Program -type Parser -constprefix Rugo rugo.ebnf`

### 3. AST Walker (`ast/walker.go`)

Walks the parse tree and produces typed AST nodes defined in `ast/nodes.go`. The AST uses Go interfaces with marker methods:

```
Node (interface)
 Statement: Program, UseStmt, ImportStmt, RequireStmt, FuncDef, TestDef,
              IfStmt, WhileStmt, ForStmt, BreakStmt, NextStmt, ReturnStmt,
              ExprStmt, AssignStmt, IndexAssignStmt
 Expr: BinaryExpr, UnaryExpr, CallExpr, IndexExpr, SliceExpr, DotExpr,
          IdentExpr, IntLiteral, FloatLiteral, StringLiteral, BoolLiteral,
          NilLiteral, ArrayLiteral, HashLiteral, TryExpr, SpawnExpr,
          ParallelExpr, FnExpr
```

Every statement embeds `BaseStmt` with `SourceLine` for `.rugo` source mapping.

### 4. Code Generation (`compiler/codegen.go`)

Converts AST nodes to Go source code. Emits:
- A `main()` function with top-level code (or TAP test harness when `rats` blocks present)
- User-defined functions as Go functions (`rugofn_NAME`)
- Module runtime code and auto-generated wrapper functions
- Shell fallback via `exec.Command("sh", "-c", ...)`
- `for..in` loops via `rugo_iterable()` / `rugo_iterable_default()`
- Index assignment via `rugo_index_set()` (type-switches arrays and hashes)
- Negative indexing via `rugo_array_index()` runtime helper
- Slicing via `rugo_slice()` for arrays and strings
- `break`/`next` as Go `break`/`continue`
- Lambdas as Go variadic anonymous functions
- `spawn` as IIFE with goroutine + `rugoTask` struct
- `parallel` as IIFE with `sync.WaitGroup` + `sync.Once`
- Task method dispatch via `rugo_task_*` runtime helpers
- Collection methods via `rugo_dot_call` runtime dispatch (`.map`, `.filter`, `.reduce`, etc.)
- `//line` directives for accurate `.rugo` source locations in panics
- Import gating: `sync`+`time` conditionally emitted based on AST flags
- Argument count validation with Rugo-friendly error messages

### Function Naming Conventions

| Rugo construct | Go function name |
|----------------|-----------------|
| `def greet(...)` | `rugofn_greet(...)` |
| `ns.func(...)` (user module) | `rugons_ns_func(...)` |
| `mod.func(...)` (stdlib module) | `rugo_mod_func(...)` |
| `puts(...)` | `rugo_puts(...)` |
| `__shell__(...)` | `rugo_shell(...)` |
| `__capture__(...)` | `rugo_capture(...)` |

## Module System

Rugo has three import mechanisms:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules (hand-crafted wrappers) | `use "http"` |
| `import` | Go stdlib bridge (auto-generated calls) | `import "strings"` |
| `require` | User `.rugo` files (local or remote) | `require "helpers"` |

### Rugo Stdlib Modules (`use`)

Modules self-register via Go `init()` using `modules.Register()`. Each module has:
- `runtime.go` — Go struct + methods (tagged `//go:build ignore`, embedded as string)
- Registration file — embeds `runtime.go`, declares function signatures with typed args

Available modules:

| Module | Description |
|--------|-------------|
| `ast` | Parse and inspect Rugo source files |
| `bench` | Benchmark framework |
| `cli` | CLI app builder with commands, flags, and dispatch |
| `color` | ANSI terminal colors and styles |
| `conv` | Type conversions |
| `fmt` | String formatting (sprintf, printf) |
| `http` | HTTP client |
| `json` | JSON parsing and encoding |
| `os` | Shell execution and process control |
| `queue` | Thread-safe queue for producer-consumer concurrency |
| `re` | Regular expressions |
| `sqlite` | SQLite database access (pure Go, no CGO) |
| `str` | String utilities |
| `test` | Testing and assertions |
| `web` | Chi-inspired HTTP router (routes, middleware, groups, static files) |

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

When `Variadic` is true on `FuncDef`, extra arguments beyond `Args` are passed as `...interface{}`.

### External Modules (Custom Rugo Builds)

You can create modules in your own Go packages and build custom Rugo binaries. Create a `main.go` that imports your modules alongside standard ones and calls `cmd.Execute(version)`. See `docs/mods.md` for examples (including `GoDeps` for third-party Go dependencies).

## Go Stdlib Bridge (`import`)

The `import` keyword gives direct access to whitelisted Go stdlib packages. The compiler generates type-safe Go calls with `interface{}` ↔ typed conversions.

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")   # HELLO
puts math.sqrt(144.0)            # 12
```

Function names use `snake_case` in Rugo, auto-mapped to Go's `PascalCase` via the registry.

### Whitelisted packages (16)

`strings`, `strconv`, `math`, `math/rand/v2`, `path/filepath`, `sort`, `os`, `time`, `encoding/json`, `encoding/base64`, `encoding/hex`, `crypto/sha256`, `crypto/md5`, `net/url`, `unicode`, `slices`, `maps`

### Key special cases

Special cases use the `Codegen` callback on `GoFuncSig` (each bridge file owns its own codegen):

- `time.Sleep` — ms→Duration conversion, void wrapped in IIFE
- `time.Now().Unix()` — method-chain calls (GoName contains `.`), int64→int cast
- `os.ReadFile` — []byte→string conversion
- `os.MkdirAll` — os.FileMode cast for permissions
- `strconv.FormatInt`/`ParseInt` — int64 conversions
- `sort.Strings`/`Ints` — copy-in/copy-out (mutate-in-place bridge)
- `filepath.Split` — tuple return mapped to Rugo array
- `filepath.Join` — variadic string args
- `encoding/json` — `rugo_json_prepare()` / `rugo_json_to_rugo()` for map type conversions
- `encoding/base64` — method-chain on `StdEncoding`/`URLEncoding`, []byte conversions
- `encoding/hex` — []byte conversions
- `crypto/sha256`/`md5` — fixed-size array → hex string via `fmt.Sprintf`
- `net/url.Parse` — struct decomposition into Rugo hash
- `unicode` — rune extraction via `rugo_utf8_decode()` helper
- `slices`, `maps` — runtime-only packages (`NoGoImport: true`), custom helper functions

### Error handling

Go `(T, error)` returns auto-panic, integrating with `try/or`:
```ruby
n = try strconv.atoi("abc") or 0
```

### Aliasing

`import "os" as go_os` when namespace conflicts with `use "os"`.

## User Modules (`require`) and Remote Modules

### Local requires
```ruby
require "helpers"            # loads helpers.rugo, namespace: helpers
require "lib/utils" as "u"  # loads lib/utils.rugo, namespace: u
```

Paths resolved relative to calling file. Directory entry points: `<dirname>.rugo` → `main.rugo` → sole `.rugo` file.

### Selective imports with `with`
```ruby
require "mylib" with client, helpers          # local directory
require "github.com/user/lib@v1.0.0" with client, issue  # remote
```

### Remote modules
```ruby
require "github.com/user/rugo-utils@v1.0.0" as "utils"
```

Cached in `~/.rugo/modules/`. Tagged versions and SHAs cached forever; branches locked to SHA on first fetch.

### Lock files

- `rugo mod tidy` — generates `rugo.lock` recording exact commit SHA for all remote modules
- `rugo mod update` — re-resolves mutable dependencies
- `rugo build --frozen` — fails if lock file is stale

See `remote/` package for implementation details.

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

RATS (Rugo Automated Testing System) tests live in `rats/` organized by category. **Load the `rugo-rats` skill** for full details.

```bash
rugo rats rats/                         # run ALL test files
rugo rats rats/core/                    # run core language tests
rugo rats rats/stdlib/                  # run stdlib module tests
rugo rats rats/gobridge/                # run Go bridge tests
rugo rats rats/tools/                   # run tool tests
rugo rats rats/core/03_control_flow_test.rugo  # run a specific test file
```

### Benchmarks

```bash
rugo bench bench/                        # run all _bench.rugo files
rugo bench bench/arithmetic_bench.rugo   # run a specific benchmark
```

Benchmark files use `use "bench"` and `bench` blocks. Auto-calibrates iterations (≥1s), reports ns/op. Output on stderr (respects `NO_COLOR`).

### Go-side Compiler Benchmarks

```bash
go test -bench=. ./compiler/ -benchmem
```

### Run the full test suite

```bash
rugo run script/test
```

### Test a .rugo script

```bash
go run . run examples/hello.rugo
go run . emit examples/hello.rugo   # inspect generated Go code
```

### Inspect generated Go for debugging

Use `emit` to see what Go code produced is this is the best way to debug codegen issues: 

```bash
go run . emit script.rugo
```

## Development Workflow

1. **Before editing**, read relevant source files and understand the pipeline stage you're modifying.
2. **Grammar changes**: Edit `rugo.ebnf`, regenerate `parser.go`, then update the walker (`ast/walker.go`) and codegen.
3. **Preprocessor changes**: Be careful with shell fallback logic — read `preference.md` for the positional resolution design. The preprocessor is in `ast/preprocess.go`.
4. **New modules**: Follow the pattern in `docs/mods.md`. Always add typed function signatures and `Doc` strings.
6. **After edits**: Run `go test ./... -count=1` and test with relevant examples.
7. **Format code**: `go fmt ./...`

## CLI Subcommands

| Command | Description |
|---------|-------------|
| `rugo run script.rugo` | Compile and run |
| `rugo build script.rugo` | Compile to native binary |
| `rugo emit script.rugo` | Print generated Go code |
| `rugo rats [path]` | Run RATS test files |
| `rugo bench [path]` | Run benchmarks |
| `rugo doc [target]` | Show documentation |
| `rugo mod tidy` | Generate lock file |
| `rugo mod update` | Re-resolve remote modules |
| `rugo tool install\|list\|remove` | Manage CLI extensions |
| `rugo dev modgen` | Scaffold a new module |

Shorthand: `rugo script.rugo` is equivalent to `rugo run script.rugo`. Installed tools are discovered as subcommands: `rugo linter ...`.

## Concurrency

Rugo has two concurrency primitives backed by goroutines. See `docs/concurrency.md` for the full design doc.

### `spawn` — single goroutine + task handle

```ruby
task = spawn
  http.get("https://example.com")
end

task = spawn http.get("https://example.com")  # one-liner sugar

task.value      # block until done, return result (re-raises panics)
task.done       # non-blocking: true if finished
task.wait(5)    # block with timeout, panics on timeout
```

### `parallel` — fan-out N expressions, wait for all

```ruby
results = parallel
  http.get("https://api1.com")
  http.get("https://api2.com")
end
puts results[0]
```

### Key implementation details

- **Task method dispatch is always-on** — `.value`/`.done`/`.wait` always compile to `rugo_task_*` helpers. The `usesTaskMethods` AST scan independently gates runtime emission.
- **Import gating:** `hasSpawn` → `sync`+`time`; `hasParallel` → `sync` only; `usesTaskMethods` → `sync`+`time`
- **One-liner limitation:** `spawn EXPR` works at line-start or after `=`, not nested in function calls

## Tools System (`rugo tool`)

Tools are compiled Rugo programs installed to `~/.rugo/tools/` that extend the CLI:

```bash
rugo tool install ./tools/linter                     # local
rugo tool install github.com/user/rugo-fmt@v1.0.0   # remote
rugo tool install core                                # all official tools
rugo tool list
rugo tool remove linter
rugo linter smart-append examples/spawn.rugo          # use installed tool
```

See `docs/tools/tool-command.md` for details. Core tools: `linter`, `fuzz`, `runmd`.

## Documentation System (`rugo doc`)

The `rugo doc` command provides introspection for `.rugo` files, stdlib modules, Go bridge packages, and remote modules.

### Architecture

| Component | Purpose |
|-----------|---------|
| `doc/doc.go` | Text-level extractor: scans raw `.rugo` source, attaches `#` comment blocks to `def`/`struct` declarations via blank-line rule |
| `doc/format.go` | Terminal formatter: renders `FileDoc`, modules, and bridge packages in `go doc`-style output |
| `cmd/cmd.go` `docAction` | CLI dispatcher: routes to file/module/bridge/remote handlers, pipes through `bat` if available |

### Doc comment convention

- Consecutive `#` lines immediately before `def`/`struct` (no blank line gap) = **doc comment**
- First `#` block at top of file (before any code) = **file-level doc**
- `#` inside function bodies, after a blank line gap, or inline = **regular comment** (not shown)

### Doc fields on registries

Both `modules.Module`/`modules.FuncDef` and `gobridge.Package`/`gobridge.GoFuncSig` have `Doc string` fields. All stdlib modules and bridge packages have populated docs. When adding new modules or bridge packages, always include `Doc` strings.

### Modes

```bash
rugo doc file.rugo              # all docs in a .rugo file
rugo doc file.rugo symbol       # specific function or struct
rugo doc http                   # stdlib module (use)
rugo doc strings                # bridge package (import)
rugo doc github.com/user/repo   # remote module
rugo doc --all                  # list all modules and bridge packages
```

### Bat integration

When `bat` is on PATH and `NO_COLOR` is not set, output is piped through `bat --language=ruby --style=plain --paging=never` for syntax highlighting.

## Common Pitfalls

* `parser.go` is generated — never edit it directly; edit `rugo.ebnf` and regenerate.
* Shell fallback is the default for unknown identifiers at top level — new builtins/keywords must be added to the preprocessor's known sets to avoid being treated as shell commands.
* Module `runtime.go` files have `//go:build ignore` tags — they're embedded as strings, not compiled directly.
* The preprocessor runs before parsing — some syntax transformations (pipes, try sugar, compound assignment, bare append) happen there, not in the parser/walker.
* AST nodes are in `ast/` package, not `compiler/` — walker, nodes, and preprocessor live there.
* Go bridge files are in `gobridge/` (top-level), not `compiler/gobridge/`.
* When adding bridge functions with special behavior, use the `Codegen` callback on `GoFuncSig` rather than adding cases to `codegen.go`.
* RATS tests are organized into `rats/core/`, `rats/stdlib/`, `rats/gobridge/`, `rats/tools/` — put new tests in the right subdirectory.
