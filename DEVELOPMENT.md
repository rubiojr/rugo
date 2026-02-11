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
| Preprocessor | `compiler/preprocess.go` | Runs before parsing. New keywords must be added here to avoid shell fallback. |
| Grammar | `parser/rugo.ebnf` | **Never** hand-edit `parser.go`. Regenerate with `egg`. |
| Walker | `compiler/walker.go` | Transforms parse tree → AST nodes (`compiler/nodes.go`). |
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

The compiler package exposes a public API for tooling (linters, formatters, refactoring tools) that need AST access without full compilation.

### Parsing

```go
c := &compiler.Compiler{}

// Parse a file into an AST without compiling to Go.
prog, err := c.ParseFile("script.rugo")

// Parse raw source code.
prog, err := c.ParseSource(`puts("hello")`, "script.rugo")
```

### AST Position Info

Every `Statement` node has `StmtLine()` (start line) and `StmtEndLine()` (end line). Block statements (`FuncDef`, `TestDef`, `BenchDef`, `IfStmt`, `WhileStmt`, `ForStmt`) span multiple lines; non-block statements have `EndLine == SourceLine`.

```go
for _, s := range prog.Statements {
    if fn, ok := s.(*compiler.FuncDef); ok {
        fmt.Printf("def %s: lines %d-%d\n", fn.Name, fn.StmtLine(), fn.StmtEndLine())
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
compiler.WalkStmts(prog, func(s compiler.Statement) {
    // called for every statement in the tree
})

// Walk all expressions. Returns true on first match.
compiler.WalkExprs(prog, func(e compiler.Expr) bool {
    _, isCall := e.(*compiler.CallExpr)
    return isCall // stops on first call expression found
})
