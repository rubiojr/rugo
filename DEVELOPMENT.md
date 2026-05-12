# Rugo Development Guide

## Build & Test

```bash
go build -o rugo .
go test ./... -count=1

# Full suite (Go tests + all examples)
rugo run script/test
```

Use `emit` to inspect generated Go code when debugging:

```bash
go run . emit script.rg
```

## Adding a New Language Feature

Every new feature **must** include:

1. **End-to-end RATS tests** (`rats/`) — this is the most important part. Add test cases to an existing `.rt` file or create a new one with fixtures in `rats/fixtures/`. Cover both happy paths and error cases.

2. **An example script** (`examples/`) — a self-contained `.rugo` file demonstrating the feature. Examples are run by `script/test` and serve as living documentation.

3. **Language docs update** (`docs/language.md`) — document the syntax, semantics, and any edge cases.

4. **Module docs** (`docs/mods.md`) — if the feature involves a new or modified module.

## Pipeline Stages

Know which stage you're modifying:

| Stage | File(s) | Notes |
|-------|---------|-------|
| Preprocessor | `ast/preprocess.go` | Runs before parsing. New keywords must be added here to avoid shell fallback. |
| Grammar | `parser/rugo.ebnf` | **Never** hand-edit `parser.go`. Regenerate with `egg`. |
| Walker | `ast/walker.go` | Transforms parse tree → AST nodes (`ast/nodes.go`). |
| Codegen | `compiler/codegen.go` | AST nodes → Go source. |

Regenerate the parser after grammar changes:

```bash
egg -o parser.go -package parser -start Program -type Parser -constprefix Rugo rugo.ebnf
```

## Modules

Follow the existing pattern in `docs/mods.md`. Each module needs:
- `modules/mymod/mymod.go` (registration)
- `modules/mymod/runtime.go` (implementation, `//go:build ignore`)
- Blank import in `main.go`

## Common Mistakes

- Forgetting to add new keywords to the preprocessor's known sets → they get treated as shell commands.
- Editing `parser.go` directly instead of `rugo.ebnf`.
- Skipping RATS tests — if it's not tested end-to-end, it's not done.

## Compiler API

The `ast` and `compiler` packages expose a public API for tooling (linters, formatters, refactoring tools) that need AST access without full compilation. AST types and parsing live in `ast/`, code generation and build orchestration live in `compiler/`.

### Parsing

```go
c := &ast.Compiler{}

// Parse a file into an AST without compiling to Go.
prog, err := c.ParseFile("script.rugo")

// Parse raw source code.
prog, err := c.ParseSource(`puts("hello")`, "script.rugo")
```

### AST Position Info

Every `Statement` node has `StmtLine()` (start line) and `StmtEndLine()` (end line). Block statements (`FuncDef`, `TestDef`, `BenchDef`, `IfStmt`, `WhileStmt`, `ForStmt`) span multiple lines; non-block statements have `EndLine == SourceLine`.

```go
for _, s := range prog.Statements {
    if fn, ok := s.(*ast.FuncDef); ok {
        fmt.Printf("def %s: lines %d-%d\n", fn.Name, fn.StmtLine(), fn.StmtEndLine())
    }
}
```

### Raw Source Access

The `Program.RawSource` field contains the original source before preprocessing (comment stripping, sugar expansion). Use it to correlate comments with AST nodes by line number:

```go
prog, _ := c.ParseFile("lib.rugo")
lines := strings.Split(prog.RawSource, "\n")
for _, s := range prog.Statements {
    if fn, ok := s.(*ast.FuncDef); ok {
        // Check the line above for a doc comment
        if fn.StmtLine() >= 2 && strings.HasPrefix(strings.TrimSpace(lines[fn.StmtLine()-2]), "#") {
            fmt.Printf("def %s has a doc comment\n", fn.Name)
        }
    }
}
```

### Type Inference

```go
ti := compiler.Infer(prog)
// ti.FuncTypes["add"].ReturnType == compiler.TypeInt
// ti.ExprType(expr) returns the inferred type of any expression
// ti.VarType("funcName", "varName") returns variable types per scope
```

### AST Traversal

```go
// Walk all statements (including nested ones inside blocks).
// Return false to skip children of the current statement.
compiler.WalkStmts(prog, func(s ast.Statement) bool {
    // called for every statement in the tree
    return true // return false to skip children
})

// Walk all expressions. Returns true on first match.
compiler.WalkExprs(prog, func(e ast.Expr) bool {
    _, isCall := e.(*ast.CallExpr)
    return isCall // stops on first call expression found
})
```

## Measuring Type Coverage & Inference Impact

Rugo emits typed Go where it can infer concrete types, falling back to
`interface{}` plus `rugo_*` runtime helpers where it cannot. Two CLI features
make this trade-off visible:

### `rugo emit --stats`

Compiles a script and prints a snapshot of how much was typed at the source
level and what shape the generated Go has:

```bash
rugo emit --stats script.rugo            # human-readable
rugo emit --stats --format json script.rugo
```

The report covers:

- **Source side** — functions (fully typed / partial / untyped), params,
  returns, locals, and expressions with typed vs dynamic counts.
- **Generated Go** — total lines, user-function count, `interface{}`
  occurrences, boxing casts (`interface{}(x)`), and `rugo_*` helper call counts
  bucketed by category (coerce, arith, compare, access, builtin, iter, method,
  shell, other).

The numbers are designed to be diffable between commits as a CI signal: more
"dynamic" and more `rugo_*` helper calls means more runtime boxing and worse
codegen.

### `--no-infer` — disable inference for A/B comparisons

`emit`, `build`, `run`, `rats`, and `bench` all accept `--no-infer`, which
forces the codegen to treat every value as `interface{}`. The inference engine
still runs (so `--stats` still reports what it would have found), but the
generated Go ignores it:

```bash
rugo emit --stats             script.rugo   # typed codegen
rugo emit --stats --no-infer  script.rugo   # untyped codegen (same source stats)

rugo bench           bench/   # current behaviour
rugo bench --no-infer bench/  # what perf looks like without inference
```

This is how you size the win from a new inference rule before/after writing
it.

### Baseline workflow (Makefile)

Use the baseline targets to measure the impact of compiler changes against a
saved older binary:

```bash
# 1. Check out the "before" revision and snapshot it
git checkout main
make baseline             # builds and copies bin/rugo -> bin/rugo-baseline

# 2. Switch to your branch and build the new compiler
git checkout my-branch
make build                # produces bin/rugo

# 3. Run both binaries against the same suite
make rats-compare         # wall-clock for full RATS suite, baseline vs current
make bench-compare        # bench suite with vs without --no-infer
make stats FILE=script.rugo   # type coverage for one script
```

`bin/rugo-baseline` is gitignored (matches `/bin/`). Re-running `make baseline`
overwrites the snapshot, so always make it from the revision you want as the
"before" reference.

Note: `rats-compare` runs the baseline first and the current build second, so
the second run benefits from a warm Go build cache. Compare runs from the same
state and use the trend across many invocations rather than a single number.

### Optional type annotations

Function params and return types can carry annotations of the form
`name : type` and `: type`. They are validated at compile time, used to seed
type inference, and (for the four primitive types `int`/`float`/`string`/`bool`)
cause codegen to emit typed Go signatures.

Internals:

| Layer | Where |
|-------|-------|
| Grammar | `parser/rugo.ebnf` — `Param`, `FuncDef`, `FnExpr` each take an optional `[ ':' ident ]`. After editing, regenerate `parser.go` with `egg`. |
| AST | `ast/nodes.go` — `Param.TypeAnnot`, `FuncDef.ReturnType`, `FnExpr.ReturnType`. |
| Walker | `ast/walker.go` — `walkParam`, `walkFuncDef`, `walkFnExpr` read the optional `:` ident token. |
| Validation | `compiler/check_annot.go` — `TypeAnnotationCheck()` rejects unknown type names. Recurses manually into `FnExpr` bodies because `visitor.WalkExprs` doesn't descend into them. |
| Seeding | `compiler/infer.go` — `Infer()` populates `FuncTypeInfo.ParamTypes`/`ReturnType` from annotations and sets `AnnotatedArgs[i]`/`AnnotatedReturn`. Annotated params are not widened by `inferFunc`; annotated returns are not overwritten. |
| Codegen | `compiler/codegen_build.go` — `coerceReturnExpr()` wraps the return value in `rugo_to_int/_float/_string/_bool` when the body is dynamic but the annotated return is typed. |
| Stats | `compiler/stats.go` — `Counter.Annotated` and `AnnotatedPct()` make annotation coverage visible in `rugo emit --stats`. |

The recognised names (`int`, `float`, `string`, `bool`, `array`, `hash`, `nil`,
`any`) are defined by `KnownTypeNames()` in `compiler/types.go`. To add a new
type name, extend `ParseTypeAnnotation` and (if it should produce a typed Go
signature) the `IsTyped`/`GoType` machinery in the same file.

When working on annotations, a space before `:` is mandatory in source — the
preprocessor's hash-colon expansion rewrites bare `ident:` into `"ident" =>`
before the parser sees it.

