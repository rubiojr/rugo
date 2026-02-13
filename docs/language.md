# Rugo Language Internals

This document describes the design and implementation of the Rugo programming language — a Ruby-inspired language that compiles to native binaries via Go.

## Overview

Rugo's compilation pipeline transforms `.rugo` source files into native binaries through a series of well-defined stages:

```
.rugo source
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

### Build Cache

During compilation, Rugo creates a temporary directory under `~/.cache/rugo/build/` to hold the generated Go source and `go.mod` before invoking `go build`. Each build gets its own uniquely-named subdirectory (`rugo-*`), which is automatically removed after compilation completes.

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

Bare `append` is also preprocessor sugar — the assignment is implicit:

```ruby
append fruits, "date"     # desugared to: fruits = append(fruits, "date")
```

Array destructuring unpacks an array into multiple variables:

```ruby
a, b, c = [10, 20, 30]   # desugared to: __destr__ = [10, 20, 30]; a = __destr__[0]; ...
```

This is preprocessor sugar. The right-hand side must be a single expression returning an array. Works with Go bridge multi-return functions:

```ruby
import "strings"
before, after, found = strings.cut("key=value", "=")
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

### Variable Scoping

Different blocks create different scoping boundaries:

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

**Functions can read top-level variables** but assigning inside a function creates a local shadow — the top-level value is not modified. Top-level variables referenced by `def` functions are promoted to package-level declarations so they are accessible. This is a key difference from lambdas, which capture the surrounding scope by reference.

**`rats` blocks are fully isolated** — they cannot see any top-level variables or constants. Use environment variables to share state between setup hooks and test blocks.

**`if` blocks are transparent** — they share the parent scope. Variables created inside an `if` block are accessible after the block ends.

**Loops create their own scope** — `while` and `for` loops can read and modify outer variables, but variables first assigned inside the loop body are local to that iteration scope. The `for` loop variable is also local.

**Lambdas capture outer scope** — they can read and modify variables from the enclosing scope. Variables assigned inside the lambda don't leak out.

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
  # body — item is value for arrays, key for hashes
end

for key, value in hash
  # body
end

for index, value in array
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

For functions with no parameters, the parentheses are optional:

```ruby
def say_hello
  puts "Hello!"
end
```

Functions are hoisted to the Go package level during codegen. Inside function bodies, all function names are visible (forward references work). At the top level, function names are only recognized after their `def` line (positional resolution).

### Lambdas (First-Class Functions)

Rugo supports anonymous functions (lambdas) using `fn(params) body end` syntax. Lambdas are first-class values — they can be stored in variables, passed as arguments, returned from functions, and stored in data structures.

```ruby
# Basic lambda
double = fn(x) x * 2 end
puts double(5)   # 10

# Multi-line lambda
classify = fn(x)
  if x > 0
    return "positive"
  end
  "non-positive"
end

# Pass lambda to function
def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end
my_map(fn(x) x * 2 end, [1, 2, 3])

# Return lambda from function (closure)
def make_adder(n)
  return fn(x) x + n end
end
add5 = make_adder(5)
puts add5(10)   # 15

# Lambdas in data structures
ops = {"add" => fn(a, b) a + b end}
puts ops["add"](2, 3)   # 5
```

Lambdas compile to Go variadic anonymous functions: `func(_args ...interface{}) interface{} { ... }`. Parameters are unpacked from the variadic args. The last expression in a lambda body is implicitly returned. Closures capture variables by reference, so mutations to captured variables are visible outside the lambda.

When a variable holding a lambda is called, the codegen emits a runtime type assertion: `variable.(func(...interface{}) interface{})(args...)`. Calling a non-function variable produces a friendly compile error: `cannot call x — not a function`.

Lambdas stored as hash values can be called via dot access, just like index access:

```ruby
ops = {
  add: fn(a, b) a + b end,
  mul: fn(a, b) a * b end
}
puts ops["add"](2, 3)   # 5 (index access)
puts ops.add(2, 3)      # 5 (dot access)
```

At runtime, `rugo_dot_call` looks up the key in the hash, type-asserts the value to a callable lambda, and invokes it. If the key doesn't exist or the value isn't a function, a friendly error is produced.

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

These are rewritten to `__capture__("...")` calls. String interpolation works inside backticks:

```ruby
name = "world"
greeting = `echo hello #{name}`   # captures "hello world"
```

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

**Limitation:** Nested double quotes inside interpolation are not supported. Use a variable instead:

```ruby
# This will NOT work:
# puts "#{h["foo"]}"

# Use a variable instead:
x = h["foo"]
puts "#{x}"
```

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

The preprocessor (`ast/preprocess.go`) runs before parsing and performs line-level source transformations. It operates in multiple passes:

### Pass 1: Compound Assignment Expansion

Desugars `+=`, `-=`, `*=`, `/=`, `%=` for both simple variables and index targets:

```
x += 1        →  x = x + 1
arr[0] -= 3   →  arr[0] = arr[0] - 3
```

### Pass 1b: Bare Append Expansion

Desugars bare `append` statements into explicit assignments. Only applies when
`append(` starts the line and the first argument is a valid assignment target:

```
append(arr, val)  →  arr = append(arr, val)
```

This pass runs after paren-free call expansion, so `append arr, val` is first
converted to `append(arr, val)`, then desugared to `arr = append(arr, val)`.

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

The preprocessor produces a line map that tracks the correspondence between preprocessed line numbers and original source line numbers. This is threaded through the walker and codegen so that `//line` directives and error messages reference the correct `.rugo` source location.

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

Statement   = UseStmt | ImportStmt | RequireStmt | SandboxStmt | FuncDef | TestDef
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

The typed AST is defined in `ast/nodes.go`. It uses Go interfaces with marker methods for type safety:

```
Node (interface)
├── Statement (interface)
│   ├── Program           — root node, contains []Statement
│   ├── UseStmt           — use "module" (Rugo stdlib)
│   ├── ImportStmt        — import "go/pkg" [as alias] (Go bridge)
│   ├── RequireStmt       — require "path" [as alias | with mod1, mod2, ...]
│   ├── SandboxStmt      — sandbox [ro: [...], rw: [...], env: [...], ...] (Landlock + env)
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
    ├── HashLiteral       — {key: value, ...} or {expr => value, ...}
    ├── TryExpr           — try expr or err handler end
    ├── SpawnExpr         — spawn body end
    ├── ParallelExpr      — parallel body end
    └── FnExpr            — fn(params) body end (lambda)
```

Every statement node embeds `BaseStmt`, which carries a `SourceLine` field mapping back to the original `.rugo` source. This is populated by the walker using the line map from the preprocessor.

### AST Walker

The walker (`ast/walker.go`) transforms the parser's flat `[]int32` encoding into the typed AST. It reads the flat array sequentially, matching non-terminal symbols to construct the appropriate node types. The walker also applies the preprocessor's line map to set accurate source line numbers on each statement.

## Code Generation

The code generator (`compiler/codegen.go`) traverses the typed AST and emits a self-contained Go `main.go` file. The generated file includes:

1. **Imports** — standard library imports plus any module-specific Go imports.
2. **Runtime helpers** — type conversion, arithmetic, comparison, shell execution, iteration, and panic handling functions.
3. **Module runtimes** — Go struct and method implementations for imported stdlib modules, plus auto-generated wrapper functions.
4. **User functions** — each `def` compiles to a Go function with signature `func rugofn_NAME(params ...interface{}) interface{}`.
5. **Main function** — top-level statements wrapped in `func main()` with a `defer/recover` for panic handling.

### Key Code Generation Patterns

**Variable scoping**: The codegen maintains a scope stack. First assignment in a scope uses `:=`, subsequent assignments use `=`. Every assigned variable gets a `_ = varname` line to suppress Go's "declared but not used" errors.

**`for..in` loops**: The single-variable form (`for x in coll`) uses `rugo_iterable_default()` which returns values for arrays and keys for hashes (Python-style). The two-variable form (`for k, v in coll`) uses `rugo_iterable()` which returns `[]rugo_kv` (key-value pairs) for uniform array/hash iteration. Arrays produce `{index, value}` pairs; hashes produce `{key, value}` pairs.

**Index assignment**: `arr[0] = x` and `hash["key"] = y` compile to `rugo_index_set(obj, idx, val)`, which type-switches on the target. Negative indices are supported for arrays (e.g., `arr[-1] = x` sets the last element).

**Negative array indexing**: Array access supports negative indices (Ruby behavior). `arr[-1]` returns the last element, `arr[-2]` the second-to-last, etc. This is handled by the `rugo_array_index` runtime helper, which normalizes negative indices by adding `len(arr)`.

**Slicing**: `obj[start, length]` compiles to `rugo_slice(obj, start, length)`, which supports both arrays and strings. For arrays it returns a new array; for strings it returns a substring. Out-of-bounds indices are clamped silently (Ruby behavior) rather than panicking. Slicing unsupported types (int, bool, hash, etc.) produces a developer-friendly error like `cannot slice hash (expected string or array)`.

**Argument count validation**: User-defined function calls are validated during code generation. If the number of arguments doesn't match the function's parameter count, a Rugo-specific error is emitted (e.g., `wrong number of arguments for greet (2 for 1)`) instead of exposing internal Go compiler errors.

**`try/or` expressions**: Compile to a Go IIFE with `defer/recover`. The tried expression is the return value; if it panics, the recovery handler runs and produces the fallback value.

**`//line` directives**: The codegen emits `//line file.rugo:N` directives before each statement so that Go runtime panics show `.rugo` source locations instead of generated Go line numbers.

**Test harness**: When `rats` blocks are present, the codegen generates a TAP-compliant test runner instead of a regular `main()`. Each test block becomes a separate function, with optional `setup`/`teardown` (per-test) and `setup_file`/`teardown_file` (per-file) hooks.

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

Rugo has three ways to bring in external functionality:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Load Rugo stdlib modules | `use "http"` |
| `import` | Bridge to Go stdlib packages | `import "strings"` |
| `require` | Load user `.rugo` files | `require "helpers"` |

### Rugo Stdlib Modules (`use`)

Modules provide namespaced standard library functionality. Each module self-registers via Go `init()` using `modules.Register()`.

A module consists of:

- **`runtime.go`** — A Go source file with a struct type and methods, tagged with `//go:build ignore` so it's not compiled directly. It's embedded as a string and emitted into the generated program.
- **Registration file** — Declares the module name, type, function signatures with typed args, required Go imports, and embeds the runtime source.

#### How Modules Work at Compile Time

1. User writes `use "http"` in their `.rugo` script.
2. The codegen looks up the module in the registry and collects its Go imports.
3. The module's `FullRuntime()` method generates:
   - The cleaned runtime source (struct + methods)
   - A module instance variable (`var _http = &HTTP{}`)
   - Wrapper functions for each declared function that convert `interface{}` args to typed parameters

#### Available Argument Types

| ArgType | Go type | Runtime converter |
|---------|---------|-------------------|
| `String` | `string` | `rugo_to_string` |
| `Int` | `int` | `rugo_to_int` |
| `Float` | `float64` | `rugo_to_float` |
| `Bool` | `bool` | `rugo_to_bool` |
| `Any` | `interface{}` | none (passed through) |

### Go Bridge (`import`)

The `import` keyword provides direct access to whitelisted Go standard library packages. The compiler maintains a static registry of bridgeable Go functions and auto-generates type conversions between Rugo's `interface{}` values and Go's typed parameters.

```ruby
import "strings"
import "math"

puts strings.contains("hello world", "world")  # true
puts math.sqrt(144.0)                           # 12
```

Function names use `snake_case` in Rugo and are auto-converted to Go's `PascalCase`. Go functions returning `(T, error)` auto-panic on error, integrating with `try/or`. The `as` keyword provides aliasing: `import "os" as go_os`.

### User Modules (`require`)

User modules use `require`:

```ruby
require "helpers"            # loads helpers.rugo, namespace: helpers
require "lib/utils" as u    # loads lib/utils.rugo, namespace: u
require "lib/utils" as "u"  # quoted form also accepted

helpers.greet("World")
u.compute(42)
```

Paths are resolved relative to the calling file. The `.rugo` extension is added automatically if missing. Requires are resolved recursively and deduplicated. If the path points to a directory, Rugo resolves an entry point: `<dirname>.rugo` → `main.rugo` → sole `.rugo` file (file takes precedence over directory when both exist).

The `with` clause selectively loads specific `.rugo` files from a directory (local or remote):

```ruby
# Local directory
require "mylib" with client, helpers
client.connect()

# Remote repository
require "github.com/user/rugo-utils@v1.0.0" with client, helpers
```

Each name loads `<name>.rugo` from the directory or repository root (falling back to `lib/<name>.rugo`), using the filename as the namespace.

Remote git repositories can also be required as a single module:

```ruby
require "github.com/user/rugo-utils@v1.0.0" as "utils"
utils.slugify("Hello World")
```

Remote modules are shallow-cloned and cached in `~/.rugo/modules/`. Tagged versions (`@v1.0.0`) and commit SHAs are cached forever; branch refs (`@main`) are locked to their resolved SHA on first fetch.

Use `rugo mod tidy` to generate a `rugo.lock` file that records the exact commit SHA for every remote module, making builds reproducible. Use `rugo mod update` to re-resolve mutable dependencies, or `rugo build --frozen` to fail if the lock file is stale.

There is no implicit search path — the require string tells you exactly where the code comes from: a relative path is local, a URL-shaped path is remote.

## Built-in Functions

These functions are always available without any `use` or `import`:

| Function | Description |
|----------|-------------|
| `puts(args...)` | Print args separated by spaces, followed by newline |
| `print(args...)` | Print args separated by spaces, no trailing newline |
| `len(v)` | Length of string, array, or hash |
| `append(arr, val)` | Append value to array, returns new array. Can be used as a bare statement: `append arr, val` |
| `raise(msg)` | Raise a runtime error with the given message |
| `type_of(v)` | Returns the type name of a value as a string |
| `exit(code?)` | Terminate the program with optional exit code (default: 0) |

## Built-in Collection Methods

Arrays and hashes have built-in methods dispatched via `rugo_dot_call`. These are always available without imports. Built-in methods take priority over hash key lookup — use `hash["key"]` for key access when a key name collides with a method.

### Array Methods

| Method | Returns | Description |
|--------|---------|-------------|
| `.map(fn)` | Array | Transform each element |
| `.filter(fn)` | Array | Keep elements where fn returns truthy |
| `.reject(fn)` | Array | Remove elements where fn returns truthy |
| `.each(fn)` | nil | Iterate with side effects |
| `.reduce(init, fn)` | Any | Accumulate: `fn(acc, val)` |
| `.find(fn)` | Any/nil | First matching element |
| `.any(fn)` | Bool | True if any element matches |
| `.all(fn)` | Bool | True if all elements match |
| `.count(fn)` | Int | Count matching elements |
| `.join(sep)` | String | Join elements with separator |
| `.first()` | Any/nil | First element |
| `.last()` | Any/nil | Last element |
| `.min()` | Any/nil | Minimum value (numeric or string) |
| `.max()` | Any/nil | Maximum value (numeric or string) |
| `.sum()` | Number | Sum of numeric elements |
| `.flatten()` | Array | Flatten one level of nesting |
| `.uniq()` | Array | Remove duplicates (preserving order) |
| `.sort_by(fn)` | Array | Sort by lambda result (non-mutating) |
| `.flat_map(fn)` | Array | Map then flatten |
| `.take(n)` | Array | First n elements |
| `.drop(n)` | Array | All but first n elements |
| `.zip(other)` | Array | Pair elements from two arrays |
| `.chunk(n)` | Array | Split into groups of n |

### Hash Methods

Hash method lambdas receive `(key, value)`:

| Method | Returns | Description |
|--------|---------|-------------|
| `.map(fn)` | Array | Transform each pair: `fn(k, v)` |
| `.filter(fn)` | Hash | Keep pairs where `fn(k, v)` returns truthy |
| `.reject(fn)` | Hash | Remove pairs where `fn(k, v)` returns truthy |
| `.each(fn)` | nil | Iterate pairs: `fn(k, v)` |
| `.reduce(init, fn)` | Any | Accumulate: `fn(acc, k, v)` |
| `.find(fn)` | Array/nil | First matching `[key, value]` pair |
| `.any(fn)` | Bool | True if any pair matches |
| `.all(fn)` | Bool | True if all pairs match |
| `.count(fn)` | Int | Count matching pairs |
| `.keys()` | Array | All keys |
| `.values()` | Array | All values |
| `.merge(other)` | Hash | Combine hashes (other wins conflicts) |

## Testing

Rugo includes a built-in test framework using `rats/end` blocks:

```ruby
use "test"

rats "arithmetic works"
  test.assert_eq(1 + 1, 2)
end

rats "string interpolation"
  name = "World"
  test.assert_eq("Hello, #{name}!", "Hello, World!")
end
```

Test files use the `_test.rugo` extension and produce TAP (Test Anything Protocol) output. The test harness supports:

- `setup` / `teardown` functions called before/after each test
- `setup_file` / `teardown_file` functions called once before/after all tests
- `test.assert_eq`, `test.assert`, `test.skip` from the test module
- Exit code 1 on any test failure

## Doc Comments

Rugo uses position-based `#` comment attachment for documentation:

```ruby
# File-level documentation goes here.

# Calculates the factorial of n.
# Returns 1 when n <= 1.
def factorial(n)
  # This is a regular comment — not shown by rugo doc
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

# A Dog with a name and breed.
struct Dog
  name
  breed
end
```

**Rules:**
- Consecutive `#` lines immediately before `def`/`struct` (no blank line gap) = **doc comment**
- First `#` block at top of file before any code = **file-level doc**
- `#` inside function bodies, after a blank line gap, or inline = **regular comment**

Use `rugo doc` to view documentation for files, modules, and bridge packages:

```bash
rugo doc file.rugo           # all docs in a file
rugo doc file.rugo factorial # specific symbol
rugo doc http              # stdlib module
rugo doc strings           # bridge package
rugo doc --all             # list everything
```
