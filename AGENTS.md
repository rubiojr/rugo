# Rugo Language Developer Agent Guide

## About Rugo

Rugo is a Ruby-inspired language that compiles to native binaries via Go.
Pipeline: `.rugo` source -> preprocess -> parse (LL(1) grammar) -> walk (typed AST) -> codegen (Go source) -> `go build` -> native binary.
Module path: `github.com/rubiojr/rugo`. Go version: 1.25.6.

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
| Embed        | `compiler/codegen_embed.go` | File embedding via `embed "path" as name` |

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

---

## Rugo Language Reference

### EBNF Grammar (LL(1))

```
Program     = { Statement | ';' } .
Statement   = UseStmt | ImportStmt | RequireStmt | EmbedStmt | SandboxStmt | FuncDef | TestDef
            | BenchDef | IfStmt | WhileStmt | ForStmt | BreakStmt | NextStmt
            | ReturnStmt | AssignOrExpr .
UseStmt     = "use" str_lit .
ImportStmt  = "import" str_lit [ "as" ident ] .
RequireStmt = "require" str_lit [ "as" ( str_lit | ident ) | "with" ident { ',' ident } ] .
EmbedStmt   = "embed" str_lit "as" ident .
FuncDef     = "def" ident '(' [ ParamList ] ')' Body "end" .
ParamList   = Param { ',' Param } .
Param       = ident [ '=' Expr ] .
Body        = { Statement | ';' } .
IfStmt      = "if" Expr Body { "elsif" Expr Body } [ "else" Body ] "end" .
CaseExpr    = "case" Expr { "of" ExprList ( "->" Expr | Body ) }
              { "elsif" Expr ( "->" Expr | Body ) } [ "else" ( "->" Expr | Body ) ] "end" .
WhileStmt   = "while" Expr Body "end" .
ForStmt     = "for" ident [ ',' ident ] "in" Expr Body "end" .
BreakStmt   = "break" .
NextStmt    = "next" .
ReturnStmt  = "return" [ Expr ] .
AssignOrExpr = Expr [ '=' Expr ] .
Expr        = OrExpr .
OrExpr      = AndExpr { "||" AndExpr } .
AndExpr     = CompExpr { "&&" CompExpr } .
CompExpr    = AddExpr [ ("==" | "!=" | "<=" | ">=" | '<' | '>') AddExpr ] .
AddExpr     = MulExpr { ('+' | '-') MulExpr } .
MulExpr     = UnaryExpr { ('*' | '/' | '%') UnaryExpr } .
UnaryExpr   = '!' Postfix | '-' Postfix | Postfix .
Postfix     = Primary { '(' [ArgList] ')' | '[' Expr [',' Expr] ']' | '.' ident } .
Primary     = ident | integer | float_lit | str_lit | raw_str_lit
            | "true" | "false" | "nil"
            | '[' [Expr {',' Expr}] ']'
            | '{' [Expr "=>" Expr {',' Expr "=>" Expr}] '}'
            | CaseExpr | TryExpr | SpawnExpr | ParallelExpr | FnExpr | '(' Expr ')' .
TryExpr      = "try" Expr "or" ident Body "end" .
SpawnExpr    = "spawn" Body "end" .
ParallelExpr = "parallel" Body "end" .
FnExpr       = "fn" '(' [ParamList] ')' Body "end" .
```

### Variables & Types

```ruby
x = 42              # integer
pi = 3.14           # float
name = "world"      # string (interpolation with #{expr})
raw = 'no #{interp}' # raw string (no interpolation, no escape processing)
flag = true         # boolean
nothing = nil       # nil
PI = 3.14           # constant (uppercase = immutable, reassignment is compile error)
```

### String Interpolation

```ruby
"Hello, #{name}! Result: #{x + 1}"
```

### Arrays & Hashes

```ruby
arr = [1, 2, 3]
arr[0]              # index (0-based)
arr[-1]             # negative index (last element)
arr[1, 2]           # slice (start, length)
arr[0] = 99         # index assignment
append arr, 4       # bare append sugar

hash = {name: "Alice", age: 30}   # colon syntax (keys become strings)
codes = {404 => "Not Found"}      # arrow syntax (any value as key)
hash["name"]                      # access
hash["email"] = "a@b.com"         # assignment
```

### Destructuring

```ruby
a, b, c = [10, 20, 30]
```

### Control Flow

```ruby
if x > 10
  puts("big")
elsif x > 5
  puts("medium")
else
  puts("small")
end

while i < 10
  i += 1
end

for item in arr
  puts(item)
end

for key, value in hash
  puts("#{key}: #{value}")
end

# case expression
result = case status
of "ok" -> "success"
of "error", "fail" -> "problem"
else -> "unknown"
end

# case with blocks
case code
of 200, 201
  puts("success")
of 404
  puts("not found")
else
  puts("other")
end
```

### Functions

```ruby
def greet(name, greeting="hello")
  return "#{greeting}, #{name}!"
end
puts greet("world")
```

### Lambdas

```ruby
double = fn(x) x * 2 end
add = fn(a, b) a + b end

transform = fn(x)
  result = x * 2
  return "value: #{result}"
end
```

### Structs

```ruby
struct Point
  x
  y
end

def Point.distance(self, other)
  dx = self.x - other.x
  dy = self.y - other.y
  return math.sqrt(dx * dx + dy * dy)
end

p = Point(10, 20)
puts p.x
```

### Error Handling (try/or)

```ruby
# Level 1: silent recovery (returns nil on failure)
result = try `might_fail`

# Level 2: default value
name = try `whoami` or "unknown"

# Level 3: handler block
result = try `cat /etc/config` or err
  puts("failed: #{err}")
  "default_value"
end
```

### Concurrency

```ruby
# spawn — single goroutine with task handle
task = spawn
  http.get("https://example.com")
end
result = task.value   # block until done
task.done             # non-blocking check
task.wait(5)          # block with timeout

# spawn one-liner
task = spawn http.get("https://api.com/data")

# parallel — fan-out, wait for all
results = parallel
  http.get("https://api1.com")
  http.get("https://api2.com")
end
puts results[0]
```

### Pipes

```ruby
echo "hello" | str.upper | puts         # shell → module → builtin
"hello world" | tr a-z A-Z | puts       # value → shell → builtin
name = echo "rugo" | str.upper          # assign pipe result
```

### Shell Commands

```ruby
`hostname`                               # backtick capture
ls -la                                   # shell fallback (unknown identifiers at top level)
output = `curl -s https://example.com`   # capture output
```

### File Embedding (compile-time)

```ruby
embed "config.yaml" as config   # baked into binary, no runtime file needed
embed "assets/logo.txt" as logo # paths relative to source file
# Cannot use ../ to escape source directory (like Go's embed)
```

### Compound Assignment

```ruby
x += 1    # also -=, *=, /=, %=
```

### Loop Control

```ruby
for item in items
  if item == "skip"
    next
  end
  if item == "stop"
    break
  end
  puts(item)
end
```

### Collection Methods (built-in on arrays and hashes)

```ruby
# Transform
arr.map(fn(x) x * 2 end)
arr.flat_map(fn(x) [x, x] end)
arr.filter(fn(x) x > 0 end)
arr.reject(fn(x) x < 0 end)
arr.reduce(0, fn(acc, x) acc + x end)

# Search
arr.find(fn(x) x > 3 end)
arr.any(fn(x) x > 4 end)
arr.all(fn(x) x > 0 end)
arr.contains(42)
arr.count(fn(x) x > 0 end)
arr.index(42)

# Sort & Order
arr.sort_by(fn(x) x end)
arr.reverse()

# Utilities
arr.first()
arr.last()
arr.min()
arr.max()
arr.sum()
arr.uniq()
arr.flatten()
arr.compact()
arr.clone()
arr.join(", ")
arr.take(3)
arr.drop(2)
arr.chunk(2)
arr.zip(other_arr)

# Hash-specific
hash.keys()
hash.values()
hash.merge(other_hash)

# Chaining
nums.filter(fn(x) x > 2 end).map(fn(x) x * 10 end).join(", ")
```

### Import System

```ruby
# use — Rugo stdlib modules
use "str"
use "http"
result = http.get("https://example.com")

# import — Go stdlib bridge
import "strings"
import "path/filepath"
puts strings.to_upper("hello")

# require — user modules
require "helpers"                                    # local file
require "lib/utils" as "u"                           # with alias
require "mylib" with client, helpers                 # selective
require "github.com/user/lib@v1.0.0" as "utils"     # remote
```

### Stdlib Modules (use "module")

| Module | Functions |
|--------|-----------|
| conv | to_i, to_f, to_s, to_bool, parse_int |
| str | contains, split, trim, starts_with, ends_with, replace, upper, lower, index, join, rune_count, count, repeat, reverse, chars, fields, trim_prefix, trim_suffix, pad_left, pad_right, each_line, center, last_index, slice, empty, byte_size |
| fmt | sprintf, printf |
| json | parse, encode, pretty |
| re | test, find, find_all, replace, replace_all, split, match |
| http | get, post, put, patch, delete |
| os | exec, exit, file_exists, is_dir, read_line, getenv, setenv, cwd, chdir, hostname, read_file, write_file, remove, mkdir, rename, glob, tmp_dir, args, pid, symlink, readlink |
| cli | name, version, about, cmd, flag, bool_flag, run, parse, command, get, args, help |
| color | red, green, yellow, blue, magenta, cyan, white, gray, bold, dim, underline, bg_* |
| web | get, post, put, delete, patch, middleware, rate_limit, group, end_group, static, listen, port, free_port, text, html, json, redirect, status |
| sqlite | open, exec, query, query_row, query_val, close |
| queue | new (then .push, .pop, .size, .close, .done) |
| math | abs, ceil, floor, round, max, min, pow, sqrt, log, log2, log10, sin, cos, tan, pi, e, clamp, random, random_int |
| rand | int, float, string, choice, shuffle, uuid |
| time | now, sleep, format, parse, since, millis |
| base64 | encode, decode, url_encode, url_decode |
| crypto | md5, sha256, sha1 |
| filepath | join, base, dir, ext, abs, rel, glob, clean, is_abs, split, match |
| hex | encode, decode |
| eval | run, file |
| test | run, tmpdir, write_file, assert_eq, assert_neq, assert_true, assert_false, assert_contains, assert_nil, fail, skip |

### Go Bridge Packages (import "package")

strings, strconv, math, math/rand/v2, path/filepath, sort, os, time, encoding/json, encoding/base64, encoding/hex, crypto/sha256, crypto/md5, net/url, unicode, slices, maps

### Builtins

| Function | Description |
|----------|-------------|
| puts(val) | Print with newline |
| print(val) | Print without newline |
| len(val) | Length of string, array, or hash |
| append(arr, v) | Append to array (or: `append arr, v`) |
| raise(msg) | Panic with message |
| type_of(val) | Returns "string", "int", "float", "bool", "nil", "array", "hash", "fn" |
| exit(code) | Exit process |

### Key Differences from Ruby

- No classes, only structs
- No blocks/procs, only fn lambdas
- No symbols, only strings
- Hash colon syntax `{key: val}` makes string keys
- Arrow syntax `{k => v}` for non-string keys
- Explicit `end` for all blocks
- `use`/`import`/`require` instead of Ruby's require/include
- Shell commands run directly (shell fallback)
- Compiles to native binary, no interpreter
- No gems/bundler; remote modules via `require "github.com/..."`
- Single `=` for assignment (no local/instance var distinction)
- Constants are uppercase identifiers (compile-time immutable)

### Example Patterns

**Read JSON file:**
```ruby
use "json"
use "os"
content = os.read_file("config.json")
data = json.parse(content)
for key, value in data
  puts "#{key}: #{value}"
end
```

**Concurrent URL fetching with spawn:**
```ruby
use "http"
def fetch_all(urls)
  tasks = []
  for url in urls
    task = spawn http.get(url)
    append tasks, task
  end
  results = []
  for t in tasks
    append results, t.value
  end
  return results
end
```

**CLI tool with flags and color:**
```ruby
use "cli"
use "color"
cli.name("greeter")
cli.version("1.0.0")
cli.about("A colorful greeting tool")
cli.flag("name", "n", "Your name", "World")
cli.bool_flag("loud", "l", "Shout the greeting", false)
cli.run(fn()
  name = cli.get("name")
  greeting = "Hello, #{name}!"
  if cli.get("loud")
    greeting = str.upper(greeting)
  end
  puts color.green(color.bold(greeting))
end)
```

**Error handling (three levels of try/or):**
```ruby
result = try `might_fail`
port = try conv.to_i(`cat /etc/port`) or 8080
config = try os.read_file("/etc/app.conf") or err
  puts "Warning: #{err}, using defaults"
  "{}"
end
```

**Web server with routes:**
```ruby
use "web"
use "json"
web.get("/", fn(req)
  web.html("<h1>Welcome to Rugo!</h1>")
end)
web.get("/api/greet/:name", fn(req)
  name = req["params"]["name"]
  web.json(json.encode({"message" => "Hello, #{name}!"}))
end)
web.listen(3000)
```

**Collection method chaining:**
```ruby
nums = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
result = nums
  .filter(fn(x) x % 2 == 0 end)
  .map(fn(x) x * x end)
  .join(", ")
puts "Even squares: #{result}"
total = nums.reduce(0, fn(acc, x) acc + x end)
people = [{"name" => "Charlie", "age" => 30}, {"name" => "Alice", "age" => 25}]
sorted = people.sort_by(fn(p) p["age"] end)
```

**File embedding (compile-time):**
```ruby
embed "config.yaml" as config
embed "templates/header.html" as header
puts config + header
```
