# Rugo Language Developer Agent Guide

## About Rugo

Rugo is a Ruby-inspired language that compiles to native binaries via Go.
Pipeline: `.rugo` source -> preprocess -> parse (LL(1) grammar) -> walk (typed AST) -> codegen (Go source) -> `go build` -> native binary.
Module path: `github.com/rubiojr/rugo`. Go version: 1.25.6.

**YOU KNOW NOTHING ABOUT RUGO. Read `docs/` before making changes.**

## Build Commands

```bash
make build                # Build bin/rugo
make test                 # Go unit tests: go test ./... -count=1
make rats                 # Build + full RATS end-to-end suite
go run . emit script.rugo # Inspect generated Go code (debugging)
```

## Test Commands

```bash
# Go unit tests (all)
go test ./... -count=1

# Single Go test
go test ./compiler/ -run TestStripComments

# RATS end-to-end tests (all) - ALWAYS run before calling any task done
make rats
# or equivalently:
bin/rugo rats --recap --timing rats/

# Single RATS test file
bin/rugo rats rats/core/02_variables_test.rugo

# Parallel RATS (faster)
bin/rugo rats --jobs 4 --recap --timing rats/

# Full suite (Go tests + all example scripts)
rugo run script/test
```

**CRITICAL**: Always run the ENTIRE `rats/` suite before calling it done.
Run with `--timing` and `--recap` to see timing info and errors summarized.

## Parser Regeneration

Never hand-edit `parser/parser.go`. Regenerate from the EBNF grammar:
```bash
egg -o parser.go -package parser -start Program -type Parser -constprefix Rugo rugo.ebnf
```

## Project Structure

| Directory    | Purpose                                                |
|------------- |------------------------------------------------------- |
| `ast/`       | AST nodes, walker, preprocessor                        |
| `parser/`    | LL(1) parser (generated from `rugo.ebnf` via `egg`)   |
| `compiler/`  | Code generation, type inference, build orchestration   |
| `modules/`   | 23 Rugo stdlib modules (registration + runtime)        |
| `gobridge/`  | Go stdlib bridge (`import` keyword)                    |
| `cmd/`       | CLI commands                                           |
| `remote/`    | Remote module fetching and lockfiles                   |
| `doc/`       | Documentation generation                               |
| `tools/`     | Linter tools                                           |
| `rats/`      | RATS end-to-end tests (~160 tests, ~410 fixtures)      |
| `examples/`  | 65+ example scripts                                    |
| `docs/`      | Language and module documentation                      |

## Pipeline Stages

| Stage        | File(s)              | Notes                                          |
|------------- |--------------------- |------------------------------------------------|
| Preprocessor | `ast/preprocess.go`  | New keywords MUST be added here to avoid shell fallback |
| Grammar      | `parser/rugo.ebnf`   | Never hand-edit `parser.go`; regenerate with `egg` |
| Walker       | `ast/walker.go`      | Transforms parse tree -> AST nodes (`ast/nodes.go`) |
| Codegen      | `compiler/codegen.go`| AST nodes -> Go source                         |

## Code Style (Go)

### Imports

Stdlib and project-internal imports are intermixed in the first group. Third-party and additional project imports go in a second group separated by a blank line:

```go
import (
    "fmt"
    "github.com/rubiojr/rugo/ast"
    "strings"

    "github.com/rubiojr/rugo/gobridge"
    "github.com/rubiojr/rugo/modules"
)
```

### Naming

- **Packages**: Short, lowercase (`ast`, `compiler`, `parser`, `gobridge`, `remote`)
- **Module packages**: Use `mod` suffix when name collides with stdlib (e.g., `package strmod`)
- **Exported types**: PascalCase (`Program`, `Statement`, `Compiler`, `SandboxConfig`)
- **Unexported types**: camelCase (`codeGen`, `walker`)
- **Interface markers**: Short lowercase methods (`node()`, `stmt()`, `expr()`)
- **Generated runtime helpers**: `rugo_` prefix (e.g., `rugo_to_bool`, `rugo_add`)
- **Constants**: Exported PascalCase (`RugoKeywords`), unexported camelCase (`rugoBuiltins`)

### Error Handling

- Standard `if err != nil` pattern with contextual `fmt.Errorf` messages
- Include file:line info in user-facing errors: `fmt.Errorf("%s:%d: ...", g.sourceFile, st.SourceLine, ...)`
- `ast.UserError` type distinguishes user errors from internal compiler errors

### Testing (Go)

- Use `testify` (`assert`/`require`) for compiler tests; raw `t.Fatal`/`t.Error` is acceptable in parser tests
- Prefer **table-driven tests** with `t.Run`:
  ```go
  tests := []struct{ name, input, expect string }{...}
  for _, tt := range tests {
      t.Run(tt.name, func(t *testing.T) { ... })
  }
  ```
- Helper functions must call `t.Helper()`
- Use `t.Cleanup` for test isolation (e.g., restoring global state)

### General

- Make Go code idiomatic, typesafe, and readable
- Rugo prefers only one way to do things -- avoid adding alternative syntax
- No linter config exists; follow standard Go conventions (`gofmt` formatting)

## Module Pattern

Every stdlib module follows a two-file structure:

1. **`modules/X/X.go`** -- Registration: uses `//go:embed runtime.go`, calls `modules.Register()` in `init()`
2. **`modules/X/runtime.go`** -- Implementation: `//go:build ignore` tag, methods return `interface{}`, pointer receivers

New modules also need a blank import in `main.go`:
```go
_ "github.com/rubiojr/rugo/modules/mymod"
```

## RATS Test Format

Tests are `.rugo` files using the `test` module:

```ruby
# RATS: Test description
use "test"

rats "test case name"
  test.assert_eq(actual, expected)
end
```

Available assertions: `test.assert_eq`, `test.assert_neq`, `test.assert_nil`,
`test.assert_true`, `test.assert_contains`. Helpers: `test.run`, `test.write_file`, `test.tmpdir`.

Fixture-based tests use `eval.run()` or `test.run()` to compile/execute `.rugo` snippets.

## Checklist for New Features

1. Add RATS tests in `rats/` (happy path + error cases)
2. Add benchmarks if performance-related
3. Add example script in `examples/`
4. Update `docs/language.md` and `docs/quickstart.md` for language features
5. Update `docs/modules.md` for new/modified modules
6. When adding doc examples, create and run the snippet to verify it works first
7. Run the full RATS suite: `make rats`

## Checklist for Bug Fixes

1. Add a regression test in `rats/`
2. If fixing a git-bug issue, close it with a detailed comment
3. Run the full RATS suite: `make rats`
