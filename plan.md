# Expression AST — Full exprString() Migration

## Problem

`exprString()` returns Go source strings. Every result gets wrapped in
`GoRawExpr{Code: ...}` at the boundary (72 sites across 5 files). This means
the expression layer is opaque — you can't inspect, test, or optimize
expressions structurally. The output AST stops at the statement level.

## Goal

Convert `exprString()` to `buildExpr()` returning `GoExpr`. All 97
`GoRawExpr{Code:}` call sites eliminated. The ENTIRE output AST is structured
from file-level down to individual expressions.

## Scope

- 910 lines in `codegen_expr.go` (17 expression methods)
- ~168 return sites producing Go expression strings
- 40 `exprString()` callers (17 in codegen_build.go, 22 in codegen_expr.go, 1 in codegen.go)
- 97 `GoRawExpr{Code:}` wrapping sites across 6 files:
  - codegen_build.go (39) — from exprString() results
  - goprint_test.go (25) — test expectations, updated as types migrate
  - codegen_expr.go (13) — internal expression wrapping
  - codegen_func.go (12) — static Go expressions in test harness
  - codegen_runtime.go (7) — static Go expressions in sandbox/runtime
  - codegen.go (1) — namespace variable value
- 5 `goExprStr`/`renderIIFE` references in codegen_expr.go + codegen_stmt.go

## New GoExpr Types (compiler/goast.go)

```go
// Literals
GoIdentExpr    { Name string }
GoIntLit       { Value string }
GoFloatLit     { Value string }
GoStringLit    { Value string }          // Go-escaped, with quotes: `"hello"`
GoBoolLit      { Value bool }
GoNilExpr      {}

// Operations  
GoBinaryExpr   { Left GoExpr; Op string; Right GoExpr }
GoUnaryExpr    { Op string; Operand GoExpr }
GoCastExpr     { Type string; Value GoExpr }           // interface{}(x), int(x)
GoTypeAssert   { Value GoExpr; Type string }           // x.(int)

// Function calls
GoCallExpr     { Func string; Args []GoExpr }          // func(args...)
GoMethodCall   { Object GoExpr; Method string;         // obj.Method(args...)
                 Args []GoExpr }
GoDotExpr      { Object GoExpr; Field string }         // obj.Field

// Collections
GoSliceLit     { Type string; Elements []GoExpr }      // []T{a, b, c}
GoMapLit       { KeyType, ValType string;              // map[K]V{k: v, ...}
                 Pairs []GoMapPair }
GoMapPair      { Key GoExpr; Value GoExpr }

// String interpolation
GoFmtSprintf   { Format string; Args []GoExpr }       // fmt.Sprintf("...", args)
GoStringConcat  { Parts []GoExpr }                     // a + b + c (all strings)

// Indexing
GoIndexExpr    { Object GoExpr; Index GoExpr }         // obj[idx]

// Parenthesized (for operator precedence)
GoParenExpr    { Inner GoExpr }                        // (expr)
```

## Printer Updates (compiler/goprint.go)

Add `exprStr(GoExpr) string` cases for every new type. The printer is the
ONLY place that converts GoExpr to text. Key formatting rules:

- `GoCallExpr{Func: "f", Args: [a, b]}` → `f(a, b)`
- `GoCastExpr{Type: "interface{}", Value: x}` → `interface{}(x)`
- `GoBinaryExpr{Left, Op: "+", Right}` → `(left + right)`
- `GoFmtSprintf{Format: "%s %d", Args: [a, b]}` → `fmt.Sprintf("%s %d", a, b)`
- `GoStringConcat{Parts: [a, b, c]}` → `a + b + c`
- `GoSliceLit{Type: "[]interface{}", Elements: [a, b]}` → `[]interface{}{a, b}`
- `GoMapLit{...}` → `map[K]V{k1: v1, k2: v2}`

## Migration Plan — 5 Phases

### Phase 1: Types + printer + buildExpr bridge

**Files:** goast.go, goprint.go, goprint_test.go, codegen_build.go

1. Add all new GoExpr types to `goast.go`
2. Add printer cases for every new type in `goprint.go`
3. Add printer tests for each type
4. Add `buildExpr(ast.Expr) (GoExpr, error)` to codegen_build.go that initially
   calls `exprString()` and wraps in `GoRawExpr` (bridge)
5. Convert all callers in `codegen_build.go` to use `buildExpr` instead of
   `exprString()` + `GoRawExpr{Code:}` manual wrapping
6. Convert callers in `codegen_func.go`, `codegen_runtime.go`, `codegen.go`
7. **Validate:** `make test && make rats` — zero behavior change

After this phase: all callers use `buildExpr()`. The bridge still returns
`GoRawExpr` internally. The new types exist but aren't used yet.

### Phase 2: Leaf expressions (literals + identifiers)

**Files:** codegen_expr.go, codegen_build.go

Convert `exprString` cases for leaf types to return structured GoExpr:

1. `IntLiteral` → `GoIntLit` (or `GoCastExpr{Type: "interface{}", Value: GoIntLit}` when boxed)
2. `FloatLiteral` → `GoFloatLit` (same boxing pattern)
3. `BoolLiteral` → `GoBoolLit` (with boxing)
4. `NilLiteral` → `GoCastExpr{Type: "interface{}", Value: GoNilExpr{}}`
5. `StringLiteral` (raw, no interpolation) → `GoStringLit` (with boxing)
6. `IdentExpr` (simple variable reference) → `GoIdentExpr`

**Approach:** Add a parallel `buildExprInner(ast.Expr) (GoExpr, error)` that
handles leaf types structurally and falls back to `exprString()` + `GoRawExpr`
for everything else. Then `buildExpr` calls `buildExprInner`.

**Boxing pattern:** Rugo boxes primitive values as `interface{}(x)`. When the
expression is typed (from type inference), it stays unboxed. This maps to:
```go
if g.exprIsTyped(e) {
    return GoIntLit{Value: ex.Value}, nil
}
return GoCastExpr{Type: "interface{}", Value: GoIntLit{Value: ex.Value}}, nil
```

**Validate:** `make test && make rats`

### Phase 3: Operators + simple calls

**Files:** codegen_expr.go

1. `BinaryExpr` (typed fast path: both operands Go-typed) → `GoParenExpr{GoBinaryExpr{...}}`
2. `BinaryExpr` (dynamic: runtime helper) → `GoCallExpr{Func: "rugo_add", Args: [left, right]}`
3. `UnaryExpr` → `GoUnaryExpr` or `GoCallExpr{Func: "rugo_negate"}`
4. `IndexExpr` → `GoCallExpr{Func: "rugo_index", Args: [obj, idx]}`
5. `SliceExpr` → `GoCallExpr{Func: "rugo_slice", Args: [obj, start, len]}`
6. `ArrayLiteral` → `GoCastExpr{Type: "interface{}", Value: GoSliceLit{...}}`
7. `HashLiteral` → `GoCastExpr{Type: "interface{}", Value: GoMapLit{...}}`

Each operator case currently has typed vs dynamic branches. Convert both branches.

**Validate:** `make test && make rats`

### Phase 4: String interpolation

**Files:** codegen_expr.go

1. `StringLiteral` with interpolation → `GoFmtSprintf` or `GoStringConcat`
   - All-string optimization path: `GoStringConcat{Parts: [a, " ", b]}`
   - General path: `GoFmtSprintf{Format: "%s %d", Args: [a, b]}`
2. `compileInterpolatedExpr` returns `GoExpr` instead of `string`

This is the trickiest conversion because interpolation has its own mini-parser
(`ast.ProcessInterpolation`) and the string optimization detects all-string
arguments to avoid `fmt.Sprintf`.

**Validate:** `make test && make rats`

### Phase 5: Complex expressions (calls, dots, lowered constructs)

**Files:** codegen_expr.go

1. `CallExpr` → `GoCallExpr` or `GoMethodCall` depending on dispatch
   - Built-in calls: `puts`, `len`, `append`, `raise`, `exit`, `type_of`, `range`
   - User function calls: `rugofn_name(args)`
   - Namespaced calls: `rugons_ns_name(args)`
   - Module method calls: module-specific codegen
   - Lambda calls: `expr.(func(...interface{}) interface{})(args)`
   This is the MOST complex case (~140 lines). Many sub-paths.

2. `DotExpr` → `GoMethodCall` or `GoCallExpr` depending on resolution
   - Module method: `module.LookupFunc()`
   - Go bridge: `gobridge.Lookup()`
   - Namespace: `rugons_ns_field`
   - Runtime: `GoCallExpr{Func: "rugo_dot_get", Args: [obj, field]}`

3. `LoweredTryExpr` → `GoIIFEExpr` (already partially done — the IIFE body
   is structured, but the outer function returns string via `goExprStr`)
4. `LoweredSpawnExpr` → same pattern
5. `LoweredParallelExpr` → same pattern
6. `FnExpr` (lambda) → custom rendering (already uses goPrinter internally)

**Key change:** These methods currently return `(string, error)`. They need to
return `(GoExpr, error)` instead. The try/spawn/parallel already build
`GoIIFEExpr` nodes — they just stringify them via `goExprStr()`. Changing the
return type makes them return the `GoIIFEExpr` directly.

**Validate:** `make test && make rats`

### Phase 6: Delete bridge + cleanup

1. Delete `exprString()` — all callers now use `buildExpr()`
2. Delete `goExprStr()` and `renderIIFE()` from codegen_stmt.go
3. Remove `GoRawExpr` usage (may keep type for edge cases)
4. Update goprint.go `exprStr()` to handle all types (remove `GoRawExpr` fallback
   or keep as safety net)
5. `go vet ./...` — clean
6. **Validate:** `make test && make rats`

## Caller Migration Checklist

Each caller of `exprString()` needs to switch to `buildExpr()`:

### codegen_build.go (17 calls)
- [ ] `buildAssign` — value expression
- [ ] `buildIndexAssign` — object, index, value
- [ ] `buildDotAssign` — object, value
- [ ] `buildExprStmt` — expression
- [ ] `buildReturn` — value
- [ ] `buildImplicitReturn` — value
- [ ] `buildTryResult` — value
- [ ] `buildSpawnReturn` — value
- [ ] `buildTryHandlerReturn` — value
- [ ] `buildIf` — condition (via condExpr)
- [ ] `buildWhile` — condition
- [ ] `buildFor` — collection
- [ ] `buildFunc` — default parameter expressions

### codegen_expr.go (22 calls — internal recursion)
- [ ] `binaryExpr` — left, right operands
- [ ] `unaryExpr` — operand
- [ ] `callExpr` — function, arguments
- [ ] `dotExpr` — object
- [ ] `indexExpr` — object, index
- [ ] `sliceExpr` — object, start, length
- [ ] `arrayLiteral` — elements
- [ ] `hashLiteral` — keys, values
- [ ] `stringLiteral` — interpolated expressions
- [ ] `loweredTryExpr` — tried expression, handler result
- [ ] `loweredSpawnExpr` — result expression
- [ ] `fnExpr` — default param expressions
- [ ] `loweredParallelExpr` — parallel branch expressions
- [ ] `compileInterpolatedExpr` — interpolated sub-expressions

### codegen_func.go (12 GoRawExpr sites — static Go expressions)
- [ ] Test harness recover/skip/fail blocks
- [ ] Passed/skipped/color assignment literals

### codegen_runtime.go (7 GoRawExpr sites — static Go expressions)
- [ ] Sandbox env checks, restrict paths/net, OS detection

### codegen.go (1 call)
- [ ] `generate()` — namespace variable values

### codegen_stmt.go (1 call)
- [ ] `renderIIFE` — result expression (will be deleted in Phase 6)

## Notes

- `exprString()` stays during migration as the internal implementation of `buildExpr()`
- Each phase progressively replaces `GoRawExpr` returns with structured types
- The printer's `exprStr(GoExpr)` method grows with each phase
- Type boxing (`interface{}(x)`) is the most common pattern — `GoCastExpr` handles it
- The `goExprStr`/`renderIIFE` in codegen_stmt.go are eliminated when
  `loweredTryExpr`/`loweredSpawnExpr`/`loweredParallelExpr` return `GoExpr` directly
- Operator precedence: binary ops wrapped in `GoParenExpr` to match current `(a + b)` output
- The `condExpr` helper (wraps in `rugo_to_bool`) becomes `buildCondExpr` returning `GoExpr`
