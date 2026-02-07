# Rugo Language Internals

This document describes the design and implementation of the Rugo programming language — a Ruby-inspired language that compiles to native binaries via Go.

## Overview

Rugo's compilation pipeline transforms `.rg` source files into native binaries through a series of well-defined stages:

```
.rg source
   │
   ▼
Strip comments
   │
   ▼
Preprocess (desugar, shell fallback, paren-free calls)
   │
   ▼
Parse (LL(1) grammar → flat AST)
   │
   ▼
Walk (flat AST → typed AST nodes)
   │
   ▼
Resolve (imports & requires)
   │
   ▼
Code generation (AST → Go source)
   │
   ▼
go build → native binary
```

The compiler is orchestrated by `compiler.Compiler`, which chains these stages together. The `run`, `build`, and `emit` CLI subcommands each exercise different parts of this pipeline.

## Language Design

### Type System

Rugo is dynamically typed. All values at runtime are Go `interface{}`. The generated Go code uses a small set of runtime helper functions (`rugo_to_bool`, `rugo_to_int`, `rugo_to_float`, `rugo_to_string`) to coerce values at the boundaries where Go requires concrete types.

Supported value types:

| Rugo type | Go representation |
|-----------|-------------------|
| Integer   | `int` |
| Float     | `float64` |
| String    | `string` |
| Boolean   | `bool` |
| Nil       | `nil` |
| Array     | `[]interface{}` |
| Hash      | `map[interface{}]interface{}` |

### Truthiness

Rugo follows Ruby-like truthiness rules: `nil` and `false` are falsy, everything else (including `0` and `""`) is truthy. This is enforced by the `rugo_to_bool` runtime function, which is used in all conditional contexts (`if`, `while`, `&&`, `||`).

### Operators

Arithmetic and comparison operators are dispatched dynamically through runtime helpers:

- **Arithmetic**: `+` (`rugo_add`), `-` (`rugo_sub`), `*` (`rugo_mul`), `/` (`rugo_div`), `%` (`rugo_mod`)
- **Comparison**: `==`, `!=`, `<`, `>`, `<=`, `>=` (all via `rugo_compare`)
- **Logical**: `&&`, `||` (short-circuit, coerced through `rugo_to_bool`)
- **Unary**: `-` (`rugo_negate`), `!` (`rugo_not`)

The `+` operator supports string concatenation: when the left operand is a string, the right operand is automatically coerced to string.

**Comparison semantics:**
- **Equality** (`==`, `!=`): Numeric coercion applies — `1 == 1.0` is `true`. Non-numeric types use strict equality.
- **Ordering** (`<`, `>`, `<=`, `>=`): Supports both numeric and string operands. Strings are compared lexicographically. Comparing incompatible types (e.g., string vs int) panics.

### Variables and Assignment

Variables are implicitly declared on first assignment. The codegen tracks declared variables per scope and emits `:=` for first assignment and `=` for subsequent ones. There are no explicit type annotations.

```ruby
x = 42          # declares x
x = x + 1       # reassigns x
```

Compound assignment operators (`+=`, `-=`, `*=`, `/=`, `%=`) are preprocessor sugar:

```ruby
x += 1          # desugared to: x = x + 1
arr[0] += 5     # desugared to: arr[0] = arr[0] + 5
```

### Constants

Identifiers starting with an uppercase letter are constants (Ruby convention). They can be assigned once but never reassigned — attempting to do so is a compile-time error.

```ruby
PI = 3.14           # constant (uppercase)
MAX_RETRIES = 5     # constant
name = "mutable"    # variable (lowercase) — can be reassigned

PI = 99             # compile error: cannot reassign constant PI
```

Constants are scoped: a constant defined inside a function is independent from one with the same name in another function or at the top level.

```ruby
MAX = 100           # top-level constant

def limit()
  MAX = 50          # separate constant, local to this function
  return MAX
end
```

Hash and array bindings declared as constants protect the binding (you can't point the name at a different value) but their contents can still be mutated:

```ruby
Config = {"host" => "localhost"}
Config["port"] = 8080   # OK — mutates contents, not the binding
Config = {}             # compile error — reassigns the binding
```

### Control Flow

Control flow uses Ruby-style `end`-delimited blocks:

```ruby
if condition
  # body
elsif other_condition
  # body
else
  # body
end

while condition
  # body
end

for item in collection
  # body
end

for key, value in hash
  # body
end
```

`break` and `next` are supported inside loops, compiling directly to Go `break` and `continue`.

### Functions

Functions are defined with `def/end` and always return `interface{}` in the generated Go:

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

def add(a, b)
  return a + b
end
```

Functions are hoisted to the Go package level during codegen. Inside function bodies, all function names are visible (forward references work). At the top level, function names are only recognized after their `def` line (positional resolution).

### Error Handling

Rugo provides three levels of error handling via `try/or`:

```ruby
# Level 1: Silent recovery (returns nil on failure)
result = try some_expression

# Level 2: Default value on failure
result = try some_expression or "default"

# Level 3: Handler block with error variable
result = try some_expression or err
  puts "caught: " + err
  "fallback"
end
```

Under the hood, `try` compiles to a Go IIFE (immediately invoked function expression) with `defer/recover`. The error is caught by Go's panic/recover mechanism, and the error message is made available as a string in the handler block.

### Shell Fallback

One of Rugo's distinctive features is shell fallback: unknown identifiers at the top level are treated as shell commands rather than producing compile errors.

```ruby
ls -la              # runs as: sh -c "ls -la"
echo "hello"        # runs as: sh -c "echo hello"
uname -a            # runs as: sh -c "uname -a"
```

The preprocessor rewrites these to `__shell__("...")` calls, which the codegen translates to `exec.Command("sh", "-c", ...)`. Shell commands inherit stdin/stdout/stderr from the parent process. Non-zero exit codes cause a panic with `rugoShellError`.

Backtick expressions capture command output instead of printing it:

```ruby
name = `whoami`     # captures output, strips trailing newline
```

These are rewritten to `__capture__("...")` calls.

### Pipe Operator

The pipe operator `|` connects expressions left-to-right, passing the output of the left side to the right side:

- **Shell command on left** → stdout is captured (like backticks)
- **Function/expression on left** → return value is used
- **Function on right** → piped value becomes the first argument
- **Shell command on right** → piped value is fed to stdin

```ruby
# Shell output → function
echo "hello world" | puts           # puts receives "hello world"

# Chaining: shell → module → builtin
echo "hello" | str.upper | puts     # prints "HELLO"

# Expression → function
len("hello") | puts                 # prints 5

# Value → shell stdin → function
"hello" | tr a-z A-Z | puts         # prints "HELLO"

# Assignment with pipe
name = echo "rugo" | str.upper      # name = "RUGO"

# Piped value prepended before existing args
echo "world" | puts "hello"         # prints "world hello"
```

**Key rules:**

- When **all** segments are shell commands (e.g. `ls | grep foo`), the line is left as a native shell pipe — backward compatible.
- Only when at least one segment is a Rugo construct (builtin, user function, module function, or expression) does pipe expansion activate.
- The `||` logical OR operator is never confused with the pipe `|`.
- Pipes inside strings (`"a | b"`) are not expanded.
- The pipe passes **return values**, not stdout output. `puts` and `print` return `nil`, so using them as a **non-final** segment in a pipe chain is a compile-time error:

```ruby
ls | puts | head        # ✗ compile error — puts returns nil, breaks the chain
ls | head | puts        # ✓ puts at the end, receives head's captured output
```

The preprocessor rewrites pipe expressions before parsing. For example, `echo "hello" | str.upper | puts` becomes `puts(str.upper(__capture__("echo \"hello\"")))`.

### String Interpolation

String interpolation uses `#{expr}` syntax inside double-quoted strings:

```ruby
name = "World"
puts "Hello, #{name}!"
puts "1 + 2 = #{1 + 2}"
```

The preprocessor handles the `#{...}` extraction, and the codegen compiles interpolated strings to `fmt.Sprintf` calls. Interpolated expressions are fully parsed through the Rugo parser to support arbitrary expressions.

### Raw Strings

Single-quoted strings are raw literals where no escape processing or interpolation happens (like Ruby's single-quoted strings):

```ruby
puts 'hello\nworld'        # prints: hello\nworld (literal backslash-n)
puts '\x1b[32mgreen'       # prints: \x1b[32mgreen (no ANSI processing)
puts 'no #{interpolation}'  # prints: no #{interpolation} (no interpolation)
```

Only two escape sequences are recognized in raw strings: `\\` (literal backslash) and `\'` (literal single quote). All other backslash sequences are kept as-is.

Raw strings are parsed by a separate `raw_str_lit` lexer rule in the grammar and produce `StringLiteral` nodes with `Raw: true`. The codegen emits these strings directly to Go string literals with appropriate escaping, bypassing the interpolation pipeline.

## Preprocessor

The preprocessor (`compiler/preprocess.go`) runs before parsing and performs line-level source transformations. It operates in multiple passes:

### Pass 1: Compound Assignment Expansion

Desugars `+=`, `-=`, `*=`, `/=`, `%=` for both simple variables and index targets:

```
x += 1        →  x = x + 1
arr[0] -= 3   →  arr[0] = arr[0] - 3
```

### Pass 2: Backtick Expansion

Converts backtick expressions to capture calls:

```
`hostname`    →  __capture__("hostname")
```

### Pass 3: Try Sugar Expansion

Expands single-line `try` forms into multi-line block form that the parser understands:

```
# try EXPR or DEFAULT expands to:
try
  EXPR
or _err
  DEFAULT
end

# try EXPR (no or) expands to:
try
  EXPR
or _err
  nil
end
```

This expansion also tracks a line map so error messages reference the original source line.

### Pass 4: Line-by-Line Processing

Each line is classified and transformed:

1. **Pipe expansion** — lines with top-level `|` (not `||`) are split into segments. If at least one segment is a Rugo construct (function/builtin/dotted ident/expression), the pipe is expanded into nested calls. All-shell pipes are left for the shell to handle natively.
2. **Keywords** (`if`, `def`, `while`, etc.) — left untouched.
3. **Assignments** (`x = ...`) — left untouched.
4. **Parenthesized calls** (`func(...)`) — left untouched.
5. **Known function, paren-free** (`puts "hi"`) — rewritten to `puts("hi")`.
6. **Unknown identifier** — rewritten to shell fallback: `__shell__("...")`.

Function name resolution is *positional* at the top level: a `def` must appear before its paren-free usage. Inside function bodies, all function names are visible (allowing forward references).

### Line Map

The preprocessor produces a line map that tracks the correspondence between preprocessed line numbers and original source line numbers. This is threaded through the walker and codegen so that `//line` directives and error messages reference the correct `.rg` source location.

## Parser

The parser is generated from an LL(1) grammar defined in `parser/rugo.ebnf` using the [egg](https://pkg.go.dev/modernc.org/egg) parser generator tool:

```
egg -o parser.go -package parser -start Program -type Parser -constprefix Rugo rugo.ebnf
```

> **Important**: `parser/parser.go` is generated code and must never be hand-edited. All grammar changes go through `rugo.ebnf`.

### Grammar Structure

The grammar defines a standard expression language with precedence levels:

```
Program     = { Statement }

Statement   = ImportStmt | RequireStmt | FuncDef | TestDef
            | IfStmt | WhileStmt | ForStmt
            | BreakStmt | NextStmt | ReturnStmt
            | AssignOrExpr

Expr        = OrExpr
OrExpr      = AndExpr { "||" AndExpr }
AndExpr     = CompExpr { "&&" CompExpr }
CompExpr    = AddExpr [ comp_op AddExpr ]
AddExpr     = MulExpr { ('+' | '-') MulExpr }
MulExpr     = UnaryExpr { ('*' | '/' | '%') UnaryExpr }
UnaryExpr   = '!' Postfix | '-' Postfix | Postfix
Postfix     = Primary { Suffix }
Suffix      = '(' [ ArgList ] ')' | '[' Expr [ ',' Expr ] ']' | '.' ident
```

Operator precedence (lowest to highest):

| Level | Operators |
|-------|-----------|
| 1     | `\|\|` |
| 2     | `&&` |
| 3     | `==` `!=` `<` `>` `<=` `>=` |
| 4     | `+` `-` |
| 5     | `*` `/` `%` |
| 6     | `!` (unary) `-` (unary) |
| 7     | `()` `[]` `.` (postfix) |

### Parser Output

The parser produces a flat `[]int32` array encoding the parse tree. Non-terminal nodes are encoded as `(-symbol, childCount, children...)` and terminal tokens as positive indices into the token stream. This compact representation is then walked by the AST walker.

## AST

The typed AST is defined in `compiler/nodes.go`. It uses Go interfaces with marker methods for type safety:

```
Node (interface)
├── Statement (interface)
│   ├── Program           — root node, contains []Statement
│   ├── ImportStmt        — import "module"
│   ├── RequireStmt       — require "path" [as "alias"]
│   ├── FuncDef           — def name(params) body end
│   ├── TestDef           — rats "name" body end
│   ├── IfStmt            — if/elsif/else/end
│   ├── WhileStmt         — while cond body end
│   ├── ForStmt           — for var [, var2] in expr body end
│   ├── BreakStmt         — break
│   ├── NextStmt          — next
│   ├── ReturnStmt        — return [expr]
│   ├── ExprStmt          — expression as statement
│   ├── AssignStmt        — target = value
│   └── IndexAssignStmt   — obj[index] = value
│
└── Expr (interface)
    ├── BinaryExpr        — left op right
    ├── UnaryExpr         — op operand
    ├── CallExpr          — func(args...)
    ├── IndexExpr         — obj[index]
    ├── SliceExpr         — obj[start, length]
    ├── DotExpr           — obj.field
    ├── IdentExpr         — variable/function reference
    ├── IntLiteral        — integer
    ├── FloatLiteral      — float
    ├── StringLiteral     — string (Raw: true for single-quoted)
    ├── BoolLiteral       — true/false
    ├── NilLiteral        — nil
    ├── ArrayLiteral      — [elem, ...]
    ├── HashLiteral       — {key => value, ...}
    └── TryExpr           — try expr or err handler end
```

Every statement node embeds `BaseStmt`, which carries a `SourceLine` field mapping back to the original `.rg` source. This is populated by the walker using the line map from the preprocessor.

### AST Walker

The walker (`compiler/walker.go`) transforms the parser's flat `[]int32` encoding into the typed AST. It reads the flat array sequentially, matching non-terminal symbols to construct the appropriate node types. The walker also applies the preprocessor's line map to set accurate source line numbers on each statement.

## Code Generation

The code generator (`compiler/codegen.go`) traverses the typed AST and emits a self-contained Go `main.go` file. The generated file includes:

1. **Imports** — standard library imports plus any module-specific Go imports.
2. **Runtime helpers** — type conversion, arithmetic, comparison, shell execution, iteration, and panic handling functions.
3. **Module runtimes** — Go struct and method implementations for imported stdlib modules, plus auto-generated wrapper functions.
4. **User functions** — each `def` compiles to a Go function with signature `func rugofn_NAME(params ...interface{}) interface{}`.
5. **Main function** — top-level statements wrapped in `func main()` with a `defer/recover` for panic handling.

### Key Code Generation Patterns

**Variable scoping**: The codegen maintains a scope stack. First assignment in a scope uses `:=`, subsequent assignments use `=`. Every assigned variable gets a `_ = varname` line to suppress Go's "declared but not used" errors.

**`for..in` loops**: Compiled using `rugo_iterable()` which returns `[]rugo_kv` (key-value pairs) for uniform array/hash iteration. Arrays produce `{index, value}` pairs; hashes produce `{key, value}` pairs.

**Index assignment**: `arr[0] = x` and `hash["key"] = y` compile to `rugo_index_set(obj, idx, val)`, which type-switches on the target. Negative indices are supported for arrays (e.g., `arr[-1] = x` sets the last element).

**Negative array indexing**: Array access supports negative indices (Ruby behavior). `arr[-1]` returns the last element, `arr[-2]` the second-to-last, etc. This is handled by the `rugo_array_index` runtime helper, which normalizes negative indices by adding `len(arr)`.

**Array slicing**: `arr[start, length]` compiles to `rugo_slice(obj, start, length)`, which returns a new array. Out-of-bounds indices are clamped silently (Ruby behavior) rather than panicking.

**Argument count validation**: User-defined function calls are validated during code generation. If the number of arguments doesn't match the function's parameter count, a Rugo-specific error is emitted (e.g., `wrong number of arguments for greet (2 for 1)`) instead of exposing internal Go compiler errors.

**`try/or` expressions**: Compile to a Go IIFE with `defer/recover`. The tried expression is the return value; if it panics, the recovery handler runs and produces the fallback value.

**`//line` directives**: The codegen emits `//line file.rg:N` directives before each statement so that Go runtime panics show `.rg` source locations instead of generated Go line numbers.

**Test harness**: When `rats` blocks are present, the codegen generates a TAP-compliant test runner instead of a regular `main()`. Each test block becomes a separate function, with optional `setup`/`teardown` hooks.

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

Modules provide namespaced standard library functionality. Each module self-registers via Go `init()` using `modules.Register()`.

### Module Structure

A module consists of:

- **`runtime.go`** — A Go source file with a struct type and methods, tagged with `//go:build ignore` so it's not compiled directly. It's embedded as a string and emitted into the generated program.
- **Registration file** — Declares the module name, type, function signatures with typed args, required Go imports, and embeds the runtime source.

### How Modules Work at Compile Time

1. User writes `import "http"` in their `.rg` script.
2. The codegen looks up the module in the registry and collects its Go imports.
3. The module's `FullRuntime()` method generates:
   - The cleaned runtime source (struct + methods)
   - A module instance variable (`var _http = &HTTP{}`)
   - Wrapper functions for each declared function that convert `interface{}` args to typed parameters

### Available Argument Types

| ArgType | Go type | Runtime converter |
|---------|---------|-------------------|
| `String` | `string` | `rugo_to_string` |
| `Int` | `int` | `rugo_to_int` |
| `Float` | `float64` | `rugo_to_float` |
| `Bool` | `bool` | `rugo_to_bool` |
| `Any` | `interface{}` | none (passed through) |

### User Modules

User modules use `require` instead of `import`:

```ruby
require "helpers"            # loads helpers.rg, namespace: helpers
require "lib/utils" as "u"  # loads lib/utils.rg, namespace: u

helpers.greet("World")
u.compute(42)
```

Required files are parsed and their `def` functions are extracted, namespaced (prefixed with the namespace), and included in the main program. Requires are resolved recursively and deduplicated.

## Built-in Functions

These functions are always available without any `import`:

| Function | Description |
|----------|-------------|
| `puts(args...)` | Print args separated by spaces, followed by newline |
| `print(args...)` | Print args separated by spaces, no trailing newline |
| `len(v)` | Length of string, array, or hash |
| `append(arr, val)` | Append value to array, returns new array |

## Testing

Rugo includes a built-in test framework using `rats/end` blocks:

```ruby
import "test"

rats "arithmetic works"
  test.assert_eq(1 + 1, 2)
end

rats "string interpolation"
  name = "World"
  test.assert_eq("Hello, #{name}!", "Hello, World!")
end
```

Test files use the `.rt` extension and produce TAP (Test Anything Protocol) output. The test harness supports:

- `setup` / `teardown` functions called before/after each test
- `test.assert_eq`, `test.assert`, `test.skip` from the test module
- Exit code 1 on any test failure
