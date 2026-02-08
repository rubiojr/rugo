---
name: rugo-quickstart
description: Rugo language quickstart guide. Load when writing .rg scripts, learning Rugo syntax, or helping users with Rugo language features.
---

# Rugo Quickstart

Get up and running with Rugo in minutes.

## Install

```
go install github.com/rubiojr/rugo@latest
```

## Run your first script

```bash
rugo run script.rg        # compile and run
rugo build script.rg      # compile to native binary
rugo emit script.rg       # print generated Go code
```

## Hello World

Create `hello.rg`:

```ruby
puts "Hello, World!"
```

Run it:

```bash
rugo run hello.rg
```

Or compile to a native binary:

```bash
rugo build hello.rg
./hello
```

`puts` prints a line. `print` does the same without a newline.

```ruby
print "Hello, "
puts "World!"
```

Comments start with `#`:

```ruby
# This is a comment
puts "not a comment"
```

## Variables

Variables are dynamically typed. No declarations needed.

```ruby
name = "Rugo"
age = 1
pi = 3.14
cool = true
nothing = nil
```

Reassignment works freely:

```ruby
x = 10
x = "now a string"
```

### Compound Assignment

```ruby
x = 10
x += 5   # 15
x -= 3   # 12
x *= 2   # 24
x /= 4   # 6
x %= 4   # 2
```

Works with strings too:

```ruby
msg = "Hello"
msg += ", World!"
puts msg
```

### Constants

Names starting with an uppercase letter are constants — they can only be assigned once:

```ruby
PI = 3.14
MAX_RETRIES = 5
AppName = "MyApp"

PI = 99   # compile error: cannot reassign constant PI
```

## Strings

Double-quoted strings support escape sequences and interpolation with `#{}`:

```ruby
name = "World"
puts "Hello, #{name}!"
```

Expressions work inside interpolation:

```ruby
x = 10
puts "#{x} squared is #{x * x}"
```

### Raw Strings

Single-quoted strings are raw — no escape processing and no interpolation:

```ruby
puts 'hello\nworld'       # prints: hello\nworld (literal, no newline)
puts 'no #{interpolation}' # prints: no #{interpolation}
```

Only `\\` (literal backslash) and `\'` (literal single quote) are recognized.

### Heredoc Strings

Heredocs are multiline string literals. Delimiters must be uppercase.

```ruby
name = "World"
html = <<HTML
<h1>Hello #{name}</h1>
<p>Welcome!</p>
HTML
```

Squiggly heredoc (`<<~`) strips common leading whitespace:

```ruby
page = <<~HTML
  <h1>Hello #{name}</h1>
  <p>Welcome!</p>
HTML
```

Raw heredoc (`<<'DELIM'`) — no interpolation:

```ruby
template = <<'CODE'
def #{method_name}
  puts "hello"
end
CODE
```

Raw squiggly heredoc (`<<~'DELIM'`) combines both.

### Concatenation

```ruby
greeting = "Hello" + ", " + "World!"
```

### String Comparison

Strings support all comparison operators with lexicographic ordering: `==`, `!=`, `<`, `>`, `<=`, `>=`.

### String Module

```ruby
use "str"

puts str.upper("hello")              # HELLO
puts str.lower("HELLO")              # hello
puts str.trim("  hello  ")           # hello
puts str.contains("hello", "ell")    # true
puts str.starts_with("hello", "he")  # true
puts str.ends_with("hello", "lo")    # true
puts str.replace("hello", "l", "r")  # herro
puts str.index("hello", "ll")        # 2

parts = str.split("a,b,c", ",")
```

## Arrays

```ruby
fruits = ["apple", "banana", "cherry"]
puts fruits[0]        # apple
puts len(fruits)      # 3
```

### Append

```ruby
fruits = append(fruits, "date")
```

### Index Assignment

```ruby
fruits[1] = "blueberry"
```

### Nested Arrays

```ruby
matrix = [[1, 2], [3, 4]]
puts matrix[0]        # [1, 2]
```

### Slicing

```ruby
numbers = [10, 20, 30, 40, 50]
first_two = numbers[0, 2]   # [10, 20]
middle    = numbers[1, 3]   # [20, 30, 40]
```

Out-of-bounds slices are clamped silently.

### Negative Indexing

```ruby
arr = [10, 20, 30, 40, 50]
puts arr[-1]    # 50 (last element)
puts arr[-2]    # 40 (second-to-last)
arr[-1] = 99
```

### Iterating

```ruby
for fruit in fruits
  puts fruit
end
```

## Hashes

Colon syntax for string keys — clean and concise:

```ruby
person = {name: "Alice", age: 30, city: "NYC"}
puts person["name"]   # Alice
puts person.name      # Alice
```

Arrow syntax for expression keys (variables, integers, booleans):

```ruby
codes = {404 => "Not Found", 500 => "Server Error"}
key = "greeting"
h = {key => "hello"}   # key is the variable value, not the string "key"
```

Both syntaxes can be mixed:

```ruby
h = {name: "Alice", 42 => "answer"}
```

### Mutation

```ruby
person["age"] = 31
person["email"] = "alice@example.com"
```

### Empty Hash

```ruby
counts = {}
counts["hello"] = 1
```

### Iterating

```ruby
for key, value in person
  puts "#{key} => #{value}"
end
```

## Control Flow

### If / Elsif / Else

```ruby
score = 85

if score >= 90
  puts "A"
elsif score >= 80
  puts "B"
else
  puts "C"
end
```

### Comparison & Logic

Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`

```ruby
if x > 0 && x < 100
  puts "in range"
end

if !done
  puts "still working"
end
```

### While

```ruby
i = 0
while i < 5
  puts i
  i += 1
end
```

## For Loops

### Array Iteration

```ruby
colors = ["red", "green", "blue"]
for color in colors
  puts color
end
```

### With Index

Two-variable form gives `index, value`:

```ruby
for i, color in colors
  puts "#{i}: #{color}"
end
```

### Hash Iteration

Two-variable form gives `key, value`:

```ruby
config = {"host" => "localhost", "port" => 3000}
for k, v in config
  puts "#{k} = #{v}"
end
```

### Break and Next

```ruby
for n in [1, 2, 3, 4, 5]
  if n == 4
    break
  end
  puts n
end
# prints 1, 2, 3

for n in [1, 2, 3, 4, 5]
  if n % 2 == 0
    next
  end
  puts n
end
# prints 1, 3, 5
```

`break` and `next` work in `while` loops too.

## Functions

### Define and Call

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

greet("World")
```

### No-Argument Functions

Functions with no parameters can omit the parentheses:

```ruby
def say_hello
  puts "Hello!"
end

say_hello
```

Both `def say_hello` and `def say_hello()` are valid.

### Return Values

```ruby
def add(a, b)
  return a + b
end

puts add(2, 3)   # 5
```

### Parenthesis-Free Calls

```ruby
puts "hello"
greet "World"
```

### Recursion

```ruby
def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

puts factorial(5)   # 120
```

## Lambdas (First-Class Functions)

Anonymous functions using `fn...end` syntax. Can be stored in variables, passed to functions, returned, and stored in data structures.

```ruby
double = fn(x) x * 2 end
puts double(5)   # 10
```

Multi-line:

```ruby
classify = fn(x)
  if x > 0
    return "positive"
  end
  return "non-positive"
end
```

Passing to functions:

```ruby
def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end

nums = my_map(fn(x) x * 2 end, [1, 2, 3])
puts nums   # [2, 4, 6]
```

Closures capture by reference:

```ruby
def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
puts add5(10)   # 15
```

Lambdas in data structures:

```ruby
ops = {
  "add" => fn(a, b) a + b end,
  "mul" => fn(a, b) a * b end
}
puts ops["add"](2, 3)   # 5
```

## Shell Commands

Unknown commands run as shell commands automatically.

```ruby
ls -la
whoami
date
```

### Pipes and Redirects

```ruby
echo "hello world" | tr a-z A-Z
echo "test" > /tmp/output.txt
echo "more" >> /tmp/output.txt
```

### Capture Output

Use backticks to capture command output into a variable:

```ruby
hostname = `hostname`
puts "Running on #{hostname}"
```

### Mix Shell and Rugo

```ruby
name = "World"
puts "Hello, #{name}!"
echo "this runs in the shell"
result = `uname -s`
puts "OS: #{result}"
```

### Shell Exit Codes

Failed shell commands exit the script immediately. Use `try` to catch failures:

```ruby
try rm /tmp/nonexistent-file
puts "still running"
```

### Pipe Operator

The `|` pipe operator connects shell commands with Rugo functions:

```ruby
use "str"
echo "hello" | str.upper | puts    # HELLO
"hello" | tr a-z A-Z | puts        # HELLO
name = echo "rugo" | str.upper
```

**Note:** The pipe passes return values, not stdout. `puts` and `print` return `nil`, so always put them at the end of a chain.

### Known Limitations

- `#` comments: Rugo strips `#` comments before shell fallback detection, so unquoted `#` in shell commands is treated as a comment. Use quotes: `echo "issue #123"`.
- Shell variable syntax: `FOO=bar` is interpreted as a Rugo assignment. Use `bash -c "FOO=bar command"` instead.

## Modules

Rugo has three module systems:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules | `use "http"` |
| `import` | Go stdlib bridge | `import "strings"` |
| `require` | User `.rg` files | `require "helpers"` |

### Rugo Stdlib Modules

```ruby
use "http"
use "conv"
use "str"

body = http.get("https://example.com")
n = conv.to_i("42")
parts = str.split("a,b,c", ",")
```

### Go Stdlib Bridge

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")   # HELLO
puts math.sqrt(144.0)            # 12
```

Function names use `snake_case` in Rugo, auto-converted to Go's `PascalCase`.

Use `as` to alias: `import "strings" as str_go`.

### Global Builtins

Available without any import: `puts`, `print`, `len`, `append`.

### User Modules

```ruby
# math_helpers.rg
def double(n)
  return n * 2
end
```

```ruby
# main.rg
require "math_helpers"
puts math_helpers.double(21)   # 42
```

Functions are namespaced by filename. User modules can `use` Rugo stdlib modules — imports are auto-propagated.

**Rules:**
- `use`, `import`, and `require` must be at the top level
- Namespaces must be unique — alias with `as` if needed
- Each module can only be imported/used once

## Error Handling

Rugo uses `try` / `or` for error handling. Three levels of control.

### Silent Recovery

```ruby
result = try `nonexistent_command`
# result is nil — script continues
```

### Default Value

```ruby
hostname = try `hostname` or "localhost"

use "conv"
port = try conv.to_i("not_a_number") or 8080
```

### Error Handler Block

```ruby
data = try `cat /missing/file` or err
  puts "Error: #{err}"
  "fallback"
end
```

## Concurrency

### spawn — single goroutine + task handle

```ruby
task = spawn
  http.get("https://example.com")
end

# One-liner sugar
task = spawn http.get("https://example.com")

# Fire-and-forget
spawn
  puts "background work"
end

# Task API
task.value      # block until done, return result
task.done       # non-blocking: true if finished
task.wait(5)    # block with timeout, panics on timeout
```

### Error Handling with spawn

```ruby
task = spawn
  http.get("https://doesnotexist.invalid")
end

body = try task.value or "request failed"
```

### parallel — fan-out, wait for all

```ruby
use "http"

results = parallel
  http.get("https://api.example.com/users")
  http.get("https://api.example.com/posts")
end

puts results[0]
puts results[1]
```

Each expression runs in its own goroutine. Results are returned in order. If any panics, `parallel` re-raises the first error — compose with `try/or`.

### Timeouts

```ruby
task = spawn `sleep 10`
result = try task.wait(2) or "timed out"
```

## Testing with RATS

RATS (Rugo Automated Testing System) uses `_test.rg` files and the `test` module.

### Writing Tests

```ruby
use "test"

rats "prints hello"
  result = test.run("rugo run greet.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end
```

### Running Tests

```bash
rugo rats                            # run all _test.rg in current dir
rugo rats test/greet_test.rg         # run a specific file
rugo rats --filter "hello"           # filter by test name
```

### Assertions

| Function | Description |
|----------|-------------|
| `test.assert_eq(a, b)` | Equal |
| `test.assert_neq(a, b)` | Not equal |
| `test.assert_true(val)` | Truthy |
| `test.assert_false(val)` | Falsy |
| `test.assert_contains(s, sub)` | String contains substring |
| `test.assert_nil(val)` | Value is nil |
| `test.fail(msg)` | Explicitly fail |

### Skipping Tests

```ruby
rats "not ready yet"
  test.skip("pending feature")
end
```

### Testing a Built Binary

```ruby
use "test"

rats "binary works"
  test.run("rugo build greet.rg -o /tmp/greet")
  result = test.run("/tmp/greet")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
  test.run("rm -f /tmp/greet")
end
```

### Inline Tests

Embed `rats` blocks in regular `.rg` files. `rugo run` ignores them; `rugo rats` executes them.

```ruby
# greet.rg
use "test"

def greet(name)
  return "Hello, " + name + "!"
end

puts greet("World")

rats "greet formats a greeting"
  test.assert_eq(greet("Rugo"), "Hello, Rugo!")
  test.assert_contains(greet("World"), "World")
end
```

```bash
rugo run greet.rg       # prints "Hello, World!" — tests ignored
rugo rats greet.rg      # runs the inline tests
```

When scanning a directory, `rugo rats` discovers both `_test.rg` files and regular `.rg` files containing `rats` blocks (directories named `fixtures` are skipped).

## Custom Modules (Advanced)

Create your own Rugo modules in Go and build a custom Rugo binary.

**runtime.go** — the Go implementation:

```go
//go:build ignore

package hello

type Hello struct{}

func (*Hello) Greet(name string) interface{} {
    return "hello, " + name
}
```

**hello.go** — module registration:

```go
package hello

import (
    _ "embed"
    "github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
    modules.Register(&modules.Module{
        Name: "hello",
        Type: "Hello",
        Funcs: []modules.FuncDef{
            {Name: "greet", Args: []modules.ArgType{modules.String}},
        },
        Runtime: modules.CleanRuntime(runtime),
    })
}
```

Build a custom Rugo binary:

```go
package main

import (
    "github.com/rubiojr/rugo/cmd"
    _ "github.com/rubiojr/rugo/modules/conv"
    _ "github.com/rubiojr/rugo/modules/http"
    // ... other standard modules ...
    _ "github.com/yourorg/rugo-hello"  // your custom module
)

func main() { cmd.Execute("v1.0.0-custom") }
```

Use in scripts:

```ruby
use "hello"
puts hello.greet("developer")   # hello, developer
```

Modules can wrap external Go libraries via `GoDeps`:

```go
modules.Register(&modules.Module{
    Name:      "slug",
    Type:      "Slug",
    Funcs:     []modules.FuncDef{{Name: "make", Args: []modules.ArgType{modules.String}}},
    GoImports: []string{`gosimpleslug "github.com/gosimple/slug"`},
    GoDeps:    []string{"github.com/gosimple/slug v1.15.0"},
    Runtime:   modules.CleanRuntime(runtime),
})
```

## Benchmarking

```ruby
use "bench"

def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

bench "fib(20)"
  fib(20)
end
```

```bash
rugo run benchmarks.rg          # run a single benchmark file
rugo bench                      # run all _bench.rg files in current dir
rugo bench bench/               # run all _bench.rg in a directory
```

The framework auto-calibrates iterations (scales until ≥1s elapsed), reports ns/op and run count.

## Go Bridge

Call Go standard library functions directly with `import`:

```ruby
import "strings"
import "math"

puts strings.to_upper("hello")                  # HELLO
puts strings.contains("hello world", "world")   # true
puts math.sqrt(144.0)                           # 12
```

### Error Handling

Go `(T, error)` returns auto-panic on error. Use `try/or`:

```ruby
import "strconv"
n = try strconv.atoi("not a number") or 0
```

### Aliasing

```ruby
use "os"
import "os" as go_os
go_os.setenv("APP", "rugo")
puts go_os.getenv("APP")
```

### Available Packages

| Package | Key Functions |
|---------|--------------|
| `strings` | contains, has_prefix, has_suffix, to_upper, to_lower, trim_space, split, join, replace, repeat, index, count, fields |
| `strconv` | atoi, itoa, format_float, parse_float, format_bool, parse_bool |
| `math` | abs, ceil, floor, round, sqrt, pow, log, max, min, sin, cos, tan |
| `path/filepath` | join, base, dir, ext, clean, is_abs, rel, split |
| `regexp` | match_string, must_compile, compile |
| `sort` | strings, ints |
| `os` | getenv, setenv, read_file, write_file, mkdir_all, remove, getwd |
| `time` | now_unix, now_nano, sleep |

## Structs

Lightweight object-oriented programming using hashes with dot access.

### Defining a Struct

```ruby
struct Dog
  name
  breed
end
```

Creates a constructor `Dog(name, breed)` plus a `new()` alias for namespaces.

### Dot Access on Hashes

```ruby
person = {"name" => "Alice", "age" => 30}
puts person.name          # Alice
person.name = "Bob"
```

Nested dot access:

```ruby
data = {"user" => {"name" => "Alice"}}
puts data.user.name       # Alice
```

### Methods

```ruby
# dog.rg
struct Dog
  name
  breed
end

def Dog.bark()
  return self.name + " says woof!"
end

def Dog.rename(new_name)
  self.name = new_name
end
```

```ruby
require "dog"

rex = dog.new("Rex", "Labrador")
puts dog.bark(rex)            # Rex says woof!
dog.rename(rex, "Rexy")
puts dog.bark(rex)            # Rexy says woof!
```

### Type Tag

```ruby
rex = Dog("Rex", "Lab")
puts rex.__type__            # Dog
```

## Web Server

Build web servers and REST APIs with the `web` module.

```ruby
use "web"

web.get("/", "home")

def home(req)
  return web.text("Hello, World!")
end

web.listen(3000)
```

### Routes and URL Parameters

Use `:name` to capture path segments:

```ruby
use "web"

web.get("/users/:id", "show_user")
web.post("/users", "create_user")

def show_user(req)
  id = req.params["id"]
  return web.json({id: id})
end

def create_user(req)
  return web.json({created: true}, 201)
end

web.listen(3000)
```

All five HTTP methods: `web.get`, `web.post`, `web.put`, `web.delete`, `web.patch`.

### The Request Object

```ruby
def my_handler(req)
  req.method        # "GET", "POST", etc.
  req.path          # "/users/42"
  req.body          # raw request body
  req.params["id"]  # URL parameters
  req.query["page"] # query string parameters
  req.header["Authorization"]  # request headers
  req.remote_addr   # client address
end
```

### Response Helpers

```ruby
web.text("hello")                    # 200 text/plain
web.text("not found", 404)           # 404 text/plain
web.html("<h1>Hi</h1>")             # 200 text/html
web.json({key: "val"})              # 200 application/json
web.json({key: "val"}, 201)         # with status code
web.redirect("/login")              # 302 redirect
web.redirect("/new", 301)           # 301 permanent
web.status(204)                     # empty response
```

### Middleware

Return `nil` to continue, or a response to stop:

```ruby
use "web"

web.middleware("require_auth")
web.get("/secret", "secret_handler")

def require_auth(req)
  if req.header["Authorization"] == nil
    return web.json({error: "unauthorized"}, 401)
  end
  return nil
end

def secret_handler(req)
  return web.text("secret data")
end

web.listen(3000)
```

Built-in middleware: `"logger"`, `"real_ip"`, `"rate_limiter"`.

Rate limiting:

```ruby
web.rate_limit(100)              # 100 requests/second per IP
web.middleware("rate_limiter")   # returns 429 when exceeded
```

Route-level middleware:

```ruby
web.get("/admin", "admin_panel", "require_auth", "require_admin")
```

### Route Groups

```ruby
web.group("/api", "require_auth")
  web.get("/users", "list_users")
  web.post("/users", "create_user")
web.end_group()
```
