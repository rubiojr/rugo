---
name: rugo-quickstart
description: Rugo language quickstart guide. Load when writing .rugo scripts, learning Rugo syntax, or helping users with Rugo language features.
---

# Rugo Quickstart

Get up and running with Rugo in minutes.

## Install

```
go install github.com/rubiojr/rugo@latest
```

## Run your first script

```bash
rugo run script.rugo        # compile and run
rugo build script.rugo      # compile to native binary
rugo emit script.rugo       # print generated Go code
rugo doc http             # show module documentation
```

## Hello World

Create `hello.rugo`:

```ruby
puts "Hello, World!"
```

Run it:

```bash
rugo run hello.rugo
```

Or compile to a native binary:

```bash
rugo build hello.rugo
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

Lowercase names remain freely reassignable. This follows Ruby convention.

### Scoping

Different blocks have different scoping rules:

**Functions have their own scope** — can read top-level variables but assigning creates a local variable:

```ruby
name = "Rugo"

def greet()
  name = "World"     # separate local variable
  return name
end

puts greet()   # World
puts name      # Rugo (unchanged)
```

**`if` blocks share the parent scope** — variables created inside are visible after:

```ruby
if true
  msg = "hello"
end
puts msg  # hello
```

**Loops create their own scope** — can read/modify outer variables, but new variables stay local:

```ruby
total = 0
for x in [1, 2, 3]
  total += x
end
puts total  # 6
# puts x    # compile error: undefined: x
```

**Lambdas capture the outer scope** (unlike functions):

```ruby
prefix = "Hello"
greet = fn(name) prefix + ", " + name end
puts greet("Rugo")  # Hello, Rugo
```

**Constants are scoped per function** — a constant in a function is independent from one at top level.

**`rats` blocks are fully isolated** — cannot see top-level variables or constants. Use environment variables to share state.

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

> **Note:** Nested double quotes inside interpolation are not supported.
> Use a variable instead: `x = h["key"]; puts "#{x}"`

### Raw Strings

Single-quoted strings are raw — no escape processing and no interpolation:

```ruby
puts 'hello\nworld'       # prints: hello\nworld (literal, no newline)
puts '\x1b[32mgreen'      # prints: \x1b[32mgreen (literal, no ANSI)
puts 'no #{interpolation}' # prints: no #{interpolation}
```

Only `\\` (literal backslash) and `\'` (literal single quote) are recognized:

```ruby
puts 'it\'s raw'          # prints: it's raw
puts 'back\\slash'        # prints: back\slash
```

Raw strings are useful for regex patterns, Windows paths, and test assertions where you need exact literal text.

### Heredoc Strings

Heredocs are multiline string literals. Delimiters must be uppercase (`[A-Z_][A-Z0-9_]*`).

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

### Slicing

Extract a substring with `text[start, length]` — same syntax as array slicing:

```ruby
text = "hello world"
puts text[0, 5]     # hello
puts text[6, 5]     # world
```

Out-of-bounds slices are clamped silently.

### Concatenation

```ruby
greeting = "Hello" + ", " + "World!"
```

Raw and double-quoted strings can be concatenated:

```ruby
puts 'raw\n' + "escaped\n"  # raw\nescaped<newline>
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
puts str.join(parts, " | ")          # a | b | c
```

## Arrays

```ruby
fruits = ["apple", "banana", "cherry"]
puts fruits[0]        # apple
puts len(fruits)      # 3
```

### Append

```ruby
append fruits, "date"
```

The explicit assignment form also works:

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

### Destructuring

Unpack an array into individual variables:

```ruby
a, b, c = [10, 20, 30]
puts a   # 10
puts b   # 20
puts c   # 30
```

Works with any expression returning an array, including Go bridge multi-return:

```ruby
import "strings"

before, after, found = strings.cut("key=value", "=")
puts before   # key
puts after    # value
puts found    # true
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

### Calling Lambdas via Dot Access

Lambdas stored in hashes can be called with dot syntax:

```ruby
ops = {
  add: fn(a, b) a + b end,
  mul: fn(a, b) a * b end
}
puts ops.add(2, 3)   # 5
puts ops.mul(4, 5)   # 20
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

Single-variable form gives **keys**:

```ruby
config = {"host" => "localhost", "port" => 3000}
for k in config
  puts k
end
# prints host, port
```

Two-variable form gives `key, value`:

```ruby
for k, v in config
  puts "#{k} = #{v}"
end
```

### Break

```ruby
for n in [1, 2, 3, 4, 5]
  if n == 4
    break
  end
  puts n
end
# prints 1, 2, 3
```

### Next

```ruby
for n in [1, 2, 3, 4, 5]
  if n % 2 == 0
    next
  end
  puts n
end
# prints 1, 3, 5
```

`break` and `next` work in `while` loops too.

## Collection Methods

Arrays and hashes have built-in methods for transforming, filtering, and querying data. No imports needed.

### Transforming

```ruby
nums = [1, 2, 3, 4, 5]

doubled = nums.map(fn(x) x * 2 end)
puts doubled    # [2, 4, 6, 8, 10]

pairs = [1, 2, 3].flat_map(fn(x) [x, x * 10] end)
puts pairs    # [1, 10, 2, 20, 3, 30]
```

### Filtering

```ruby
nums = [1, 2, 3, 4, 5]

big = nums.filter(fn(x) x > 3 end)
puts big    # [4, 5]

small = nums.reject(fn(x) x > 3 end)
puts small    # [1, 2, 3]
```

### Reducing

```ruby
nums = [1, 2, 3, 4, 5]

sum = nums.reduce(0, fn(acc, x) acc + x end)
puts sum    # 15

puts nums.sum()    # 15
```

### Searching

```ruby
nums = [1, 2, 3, 4, 5]

found = nums.find(fn(x) x > 3 end)
puts found    # 4

puts nums.any(fn(x) x > 4 end)    # true
puts nums.all(fn(x) x > 0 end)    # true
puts nums.count(fn(x) x > 2 end)    # 3
```

### Utilities

```ruby
words = ["hello", "world", "rugo"]

puts words.join(", ")    # hello, world, rugo
puts words.first()       # hello
puts words.last()        # rugo
puts [3, 1, 4, 1, 5].min()    # 1
puts [3, 1, 4, 1, 5].max()    # 5
puts [1, 2, 2, 3, 1].uniq()    # [1, 2, 3]
puts [[1, 2], [3, 4]].flatten()    # [1, 2, 3, 4]
puts ["banana", "fig", "apple"].sort_by(fn(s) len(s) end)
# [fig, apple, banana]
```

### Slicing

```ruby
nums = [1, 2, 3, 4, 5]

puts nums.take(3)    # [1, 2, 3]
puts nums.drop(3)    # [4, 5]
puts nums.chunk(2)    # [[1, 2], [3, 4], [5]]
puts [1, 2, 3].zip(["a", "b", "c"])    # [[1, a], [2, b], [3, c]]
```

### Chaining

Methods return arrays, so they chain naturally:

```ruby
result = [1, 2, 3, 4, 5]
  .filter(fn(x) x > 2 end)
  .map(fn(x) x * 10 end)
  .join(" + ")
puts result    # 30 + 40 + 50
```

### Hash Methods

Hash methods pass `(key, value)` to lambdas:

```ruby
person = {name: "Alice", age: 30, city: "NYC"}

puts person.map(fn(k, v) "#{k}=#{v}" end)

adults = {alice: 30, bob: 17, carol: 25}
  .filter(fn(k, v) v >= 18 end)
puts adults    # {alice: 30, carol: 25}

found = person.find(fn(k, v) v == 30 end)
puts found    # [age, 30]

puts person.keys()
puts person.values()

merged = person.merge({email: "alice@test.com"})

total = {a: 10, b: 20}.reduce(0, fn(acc, k, v) acc + v end)
puts total    # 30

puts person.any(fn(k, v) v == 30 end)    # true
puts person.count(fn(k, v) type_of(v) == "String" end)    # 2
```

### Each

Use `each` for iteration with side effects:

```ruby
items = []
[1, 2, 3].each(fn(x)
  items = append(items, x * 10)
end)
puts items    # [10, 20, 30]
```

> **Note:** `for..in` is the primary loop form. Use `each` when you need
> a functional style or want to pass iteration as a callback.

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

Closures capture by reference — changes to the outer variable are visible:

```ruby
x = 10
f = fn() x end
x = 20
puts f()   # 20
```

Closures can also mutate captured variables:

```ruby
def make_counter()
  count = 0
  inc = fn()
    count = count + 1
    return count
  end
  return inc
end

counter = make_counter()
puts counter()   # 1
puts counter()   # 2
```

Returning closures:

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

Calling via dot access:

```ruby
ops = {
  add: fn(a, b) a + b end,
  mul: fn(a, b) a * b end
}
puts ops.add(2, 3)      # 5
puts ops.mul(4, 5)      # 20
```

Composing lambdas:

```ruby
compose = fn(f, g)
  return fn(x) f(g(x)) end
end

double = fn(x) x * 2 end
inc = fn(x) x + 1 end
double_then_inc = compose(inc, double)
puts double_then_inc(5)   # 11
```

## Shell Commands

Unknown commands run as shell commands automatically.

```ruby
ls -la
whoami
date
```

### Pipes

```ruby
echo "hello world" | tr a-z A-Z
ls -la | head -5
```

### Redirects

```ruby
echo "test" > /tmp/output.txt
echo "more" >> /tmp/output.txt
ls /nonexistent 2>/dev/null
```

### Capture Output

Use backticks to capture command output into a variable:

```ruby
hostname = `hostname`
puts "Running on #{hostname}"
```

Backticks run the command and return stdout as a trimmed string. String interpolation works inside backticks:

```ruby
name = "world"
greeting = `echo hello #{name}`
puts greeting   # hello world
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

**Note:** The pipe passes **return values**, not stdout. `puts` and `print` return `nil`, so using them in the middle of a chain is a compile error — always put them at the end:

```ruby
ls | puts | head        # ✗ compile error
ls | head | puts        # ✓ puts at the end
```

### Known Limitations

- `#` comments: Rugo strips `#` comments before shell fallback detection, so unquoted `#` in shell commands is treated as a comment. Use quotes: `echo "issue #123"`.
- Shell variable syntax: `FOO=bar` is interpreted as a Rugo assignment. Use `bash -c "FOO=bar command"` instead.

## Modules

Rugo has three module systems:

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules | `use "http"` |
| `import` | Go stdlib bridge | `import "strings"` |
| `require` | User `.rugo` files | `require "helpers"` |

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

Available without any import: `puts`, `print`, `len`, `append`, `exit`, `raise`, `type_of`.

`exit` terminates the program with an optional exit code (defaults to 0):

```ruby
exit        # exit with code 0
exit(1)     # exit with code 1
```

### User Modules

```ruby
# math_helpers.rugo
def double(n)
  return n * 2
end
```

```ruby
# main.rugo
require "math_helpers"
puts math_helpers.double(21)   # 42
```

Functions are namespaced by filename. User modules can `use` Rugo stdlib modules — imports are auto-propagated. Paths are resolved relative to the calling file.

### Directory Modules

If the require path points to a directory, Rugo resolves an entry point:

1. `<dirname>.rugo` (e.g., `mylib/mylib.rugo`)
2. `main.rugo`
3. The sole `.rugo` file (if there's exactly one)

If a file `mylib.rugo` exists alongside a directory `mylib/`, the file takes precedence.

### Multi-File Libraries with `with`

Use `with` to selectively load specific `.rugo` files from a local directory:

```ruby
require "mylib" with greet, math

puts greet.greet("world")
puts math.double(21)   # 42
```

Each name in the `with` list loads `<name>.rugo` from the directory root, or from `lib/<name>.rugo` as a fallback. The filename becomes the namespace. Works with both local directories and remote repositories.

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

Fire and forget:

```ruby
try nonexistent_command
puts "still running"
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

### Raising Errors

Use `raise` to signal errors. Works like Go's `panic()` — caught with `try/or`:

```ruby
raise("something went wrong")
raise "something went wrong"      # paren-free
```

Use in functions to validate inputs:

```ruby
def greet(name)
  if name == nil
    raise "name is required"
  end
  return "Hello, " + name
end

msg = try greet(nil) or err
  puts "Error: " + err
  "Hello, stranger"
end
```

Called without arguments, `raise` uses a default message (`"runtime error"`).

## Concurrency

### Fire and Forget

```ruby
spawn
  puts "working in background"
end

puts "main continues immediately"
```

### Getting Results

```ruby
use "http"

task = spawn
  http.get("https://httpbin.org/get")
end

body = task.value
puts body
```

### One-Liner Form

```ruby
task = spawn http.get("https://httpbin.org/get")
puts task.value
```

### Task API

```ruby
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

### Queues

For producer-consumer patterns, use the `queue` module:

```ruby
use "queue"
use "conv"

q = queue.new()

spawn
  for i in [1, 2, 3]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts conv.to_s(item)
end)
```

Queues support bounded capacity (`queue.new(10)`), pop with timeout (`try q.pop(5) or "timeout"`), and properties (`q.size`, `q.closed`).

## Testing with RATS

RATS (Rugo Automated Testing System) uses `_test.rugo` files and the `test` module.

### Writing Tests

```ruby
use "test"

rats "prints hello"
  result = test.run("rugo run greet.rugo")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end
```

### Running Tests

```bash
rugo rats                       # run all _test.rugo files in rats/ (or current dir)
rugo rats test/greet_test.rugo  # run a specific file
rugo rats --filter "hello"      # filter by test name
rugo rats --timing              # show per-test and total elapsed time
rugo rats --recap               # print all failures with details at the end
```

### Capturing Command Output

`test.run(cmd)` returns a hash with:

- `"status"` — exit code (integer)
- `"output"` — combined stdout+stderr (string)
- `"lines"` — output split by newlines (array)

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
  test.run("rugo build greet.rugo -o /tmp/greet")
  result = test.run("/tmp/greet")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
  test.run("rm -f /tmp/greet")
end
```

### Setup and Teardown

| Hook | Scope | When it runs |
|------|-------|-------------|
| `def setup_file()` | Per file | Once before all tests in the file |
| `def teardown_file()` | Per file | Once after all tests in the file |
| `def setup()` | Per test | Before each individual test |
| `def teardown()` | Per test | After each individual test |

```ruby
use "test"
use "os"

def setup_file()
  os.exec("mkdir -p /tmp/myapp_test")
end

def teardown_file()
  os.exec("rm -rf /tmp/myapp_test")
end

def setup()
  test.write_file(test.tmpdir() + "/input.txt", "default")
end
```

`teardown_file()` always runs, even if tests fail.

### Inline Tests

Embed `rats` blocks in regular `.rugo` files. `rugo run` ignores them; `rugo rats` executes them.

```ruby
# greet.rugo
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
rugo run greet.rugo       # prints "Hello, World!" — tests ignored
rugo rats greet.rugo      # runs the inline tests
```

When scanning a directory, `rugo rats` discovers both `_test.rugo` files and regular `.rugo` files containing `rats` blocks (directories named `fixtures` are skipped).

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
rugo run benchmarks.rugo          # run a single benchmark file
rugo bench                      # run all _bench.rugo files in current dir
rugo bench bench/               # run all _bench.rugo in a directory
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

### Type Conversions

```ruby
import "strconv"

n = strconv.atoi("42")           # string → int
s = strconv.itoa(42)             # int → string
f = strconv.parse_float("3.14")  # string → float
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

### Multi-Return Functions

Go functions returning multiple values are bridged as arrays. Use destructuring:

```ruby
import "strings"

before, after, found = strings.cut("key=value", "=")
puts before   # key
puts after    # value
puts found    # true
```

### Available Packages

| Package | Key Functions |
|---------|--------------|
| `strings` | contains, has_prefix, has_suffix, to_upper, to_lower, trim_space, split, join, replace, repeat, index, count, fields, contains_func, index_func, map |
| `strconv` | atoi, itoa, format_float, parse_float, format_bool, parse_bool |
| `math` | abs, ceil, floor, round, sqrt, pow, log, max, min, sin, cos, tan |
| `path` | base, clean, dir, ext, is_abs, join, match, split |
| `path/filepath` | join, base, dir, ext, clean, is_abs, rel, split |
| `sort` | strings, ints |
| `os` | getenv, setenv, read_file, write_file, mkdir_all, remove, getwd |
| `time` | now_unix, now_nano, sleep |
| `encoding/json` | marshal, unmarshal, marshal_indent |
| `encoding/base64` | encode, decode, url_encode, url_decode |
| `encoding/hex` | encode, decode |
| `crypto/sha256` | sum256 |
| `crypto/md5` | sum |
| `net/url` | parse, path_escape, path_unescape, query_escape, query_unescape |
| `unicode` | is_letter, is_digit, is_space, is_upper, is_lower, is_punct, to_upper, to_lower |
| `html` | escape_string, unescape_string |
| `slices` | contains, index, reverse, compact |
| `maps` | keys, values, clone, equal |

### JSON

```ruby
import "encoding/json"

data = {name: "Rugo", version: 1}
text = json.marshal(data)
puts text                        # {"name":"Rugo","version":1}

parsed = json.unmarshal(text)
puts parsed.name                 # Rugo
```

### Encoding (Base64 & Hex)

```ruby
import "encoding/base64"
import "encoding/hex"

b64 = base64.encode("Hello!")
puts base64.decode(b64)          # Hello!

h = hex.encode("Hello!")
puts hex.decode(h)               # Hello!
```

### Hashing

```ruby
import "crypto/sha256"
import "crypto/md5"
import "encoding/hex"

puts hex.encode(sha256.sum256("hello"))   # SHA-256 hex digest
puts hex.encode(md5.sum("hello"))         # MD5 hex digest
```

### URL Parsing

```ruby
import "net/url"

u = url.parse("https://example.com:8080/path?q=hello#top")
puts u.scheme     # https
puts u.hostname   # example.com
puts u.port       # 8080
puts u.path       # /path
puts u.query      # q=hello
puts u.fragment   # top
```

### Collections (Slices & Maps)

```ruby
import "slices"
import "maps"

puts slices.contains(["a", "b", "c"], "b")  # true
puts slices.reverse([1, 2, 3])               # [3, 2, 1]

h = {name: "Rugo", lang: "go"}
puts maps.keys(h)                # [lang, name]
copy = maps.clone(h)
puts maps.equal(h, copy)         # true
```

Use `rugo doc <package>` to see all functions with typed signatures and documentation.

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
# dog.rugo
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

### Type Introspection

Use `type_of()` to get the type name of any value. For structs, it returns the struct name:

```ruby
rex = Dog("Rex", "Lab")
puts type_of(rex)            # Dog
puts type_of("hello")        # String
puts type_of(42)             # Integer
puts type_of([1, 2])         # Array
puts type_of({a: 1})         # Hash
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
  return web.json({"id" => id})
end

def create_user(req)
  return web.json({"created" => true}, 201)
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
web.json({"key" => "val"})          # 200 application/json
web.json({"key" => "val"}, 201)     # with status code
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
    return web.json({"error" => "unauthorized"}, 401)
  end
  return nil
end

def secret_handler(req)
  return web.text("secret data")
end

web.listen(3000)
```

Built-in middleware: `"logger"`, `"real_ip"`, `"rate_limiter"`.

`real_ip` resolves client IP from proxy headers (`X-Forwarded-For`, `X-Real-Ip`).

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

## Remote Modules

Load `.rugo` modules directly from git repositories — no package registry needed.

### Basic Usage

```ruby
require "github.com/user/my-utils@v1.0.0" as "utils"
puts utils.slugify("Hello World")
```

### Version Pinning

| Syntax | Meaning |
|--------|---------|
| `@v1.2.0` | Git tag (cached forever) |
| `@main` | Branch (re-fetched each build) |
| `@abc1234` | Commit SHA (cached forever) |
| *(none)* | Default branch (re-fetched) |

### Multi-File Libraries with `with`

Works with both remote repositories and local directories:

```ruby
require "github.com/rubiojr/rugh@v1.0.0" with client, issue

gh = client.from_env()
issues = issue.list(gh, "rubiojr", "rugo")
```

Each name loads `<name>.rugo` from the repo root (or `lib/<name>.rugo` as fallback). Without `with`, Rugo looks for `<repo-name>.rugo`, then `main.rugo`, then the sole `.rugo` file.

### Publishing a Multi-File Module

Publishing is just pushing a git repo. No registry, no manifest.

```
my-lib/
  client.rugo       # → client namespace
  helpers.rugo      # → helpers namespace
  main.rugo         # (optional) entry point for bare require
```

Rules for module authors:
- Each `.rugo` file at the repo root becomes a loadable module
- Functions prefixed with `_` are **private** — compiler rejects external calls
- Add a `main.rugo` if you want `require "..."` (without `with`) to work

### Inter-Module Dependencies

If one module calls functions from another, the consumer must load both:

```ruby
require "github.com/user/lib@v1.0.0" with client, issue
# client is loaded first, so issue can call client.get()
```

Order matters — load dependencies before modules that use them.

### Subpath Requires

```ruby
require "github.com/user/lib/client@v1.0.0"
```

### Cache

Remote modules are cached in `~/.rugo/modules/`. Override with `RUGO_MODULE_DIR`.

### Lock File (`rugo.lock`)

Use `rugo mod tidy` to generate a lock file that pins exact commit SHAs:

```bash
rugo mod tidy                              # resolve and write rugo.lock
rugo mod update                            # re-resolve all mutable deps
rugo mod update github.com/user/repo       # re-resolve a specific module
rugo build --frozen app.rugo -o app        # fail if lock is missing/stale
```

Format: `<module-path> <version-label> <resolved-sha>` per line.

Best practices:
- Commit `rugo.lock` for reproducible builds
- Use `--frozen` in CI to catch unintentional dependency changes
- Run `rugo mod tidy` in each directory with remote dependencies

## Doc Comments

Rugo uses `#` comments for documentation. Write doc comments immediately before `def` or `struct` declarations with no blank line gap.

### Convention

```ruby
# File-level documentation goes here.

# Calculates the factorial of n.
# Returns 1 when n <= 1.
def factorial(n)
  # This is a regular comment — NOT shown by rugo doc
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

### `rugo doc` Command

```bash
rugo doc file.rugo              # all docs in a file
rugo doc file.rugo factorial    # specific function or struct
rugo doc http                 # stdlib module
rugo doc strings              # bridge package
rugo doc use:os               # disambiguate: force stdlib module
rugo doc import:os            # disambiguate: force bridge package
rugo doc github.com/user/repo # remote module
rugo doc --all                # list all modules and packages
```

When `bat` is installed, output is syntax-highlighted automatically. Set `NO_COLOR=1` to disable.

## Sandbox

Opt-in process sandboxing using Linux Landlock. Restrict filesystem paths and network ports.

### Basic Usage

```ruby
# Deny everything (maximum restriction)
sandbox

# Allow specific paths
sandbox ro: ["/etc"], rw: ["/tmp"], rox: ["/usr/bin"]

# Allow network
sandbox connect: [80, 443], bind: 8080
```

### Permission Types

| Keyword | Access | Example |
|---------|--------|---------|
| `ro` | Read-only | Config files |
| `rw` | Read + write | Temp/output dirs |
| `rox` | Read + execute | Binary dirs |
| `rwx` | Read + write + execute | Plugin dirs |
| `connect` | TCP connect | HTTP clients |
| `bind` | TCP bind | Servers |
| `env` | Env var allowlist | Restrict env access |

### CLI Flags

Apply sandbox restrictions without modifying the script:

```bash
rugo run --sandbox --ro /etc --rox /usr --connect 443 --env PATH script.rugo
```

CLI flags **override** any `sandbox` directive in the script.

### Important Notes

- **Linux only**: On other platforms, the directive is a no-op with a warning.
- **No auto-allows**: You must explicitly allow every path.
- Shell commands typically need `rox: ["/usr", "/lib"]` and `rw: ["/dev/null"]`.
- `stat()` is not restricted: `os.file_exists()` always works regardless of sandbox.

### Environment Variable Filtering

Restrict which env vars are visible (opt-in, works on all platforms):

```ruby
sandbox env: ["PATH", "HOME"]
import "os"
puts(os.getenv("HOME"))   # works
puts(os.getenv("SECRET"))  # empty string
```

## Go Modules via Require (Advanced)

Rugo can `require` Go packages directly — the compiler introspects Go source, discovers exported functions, and bridges them automatically. No manifest, no registration boilerplate.

### Creating a Go Module

```
greeter/
  go.mod
  greeter.go
```

**greeter.go:**

```go
package greeter

import "strings"

func Hello(name string) string {
    return "Hello, " + name + "!"
}

func Shout(text string) string {
    return strings.ToUpper(text) + "!"
}
```

### Using It from Rugo

```ruby
require "greeter"

puts(greeter.hello("World"))    # Hello, World!
puts(greeter.shout("hello"))    # HELLO!
```

Function names are auto-converted: `Hello` → `hello`, `IsEmpty` → `is_empty`.

### Module Naming

The namespace is derived from the last segment of the require path. Use `as` to override:

```ruby
require "greeter" as g
puts(g.hello("Alias"))
```

### Remote Go Modules

```ruby
require "github.com/user/rugo-greeter@v1.0.0" as greeter
puts(greeter.hello("Remote"))
```

### Supported Types

| Go type | Rugo type | Notes |
|---------|-----------|-------|
| `string` | string | |
| `int` | integer | |
| `float64` | float | |
| `bool` | boolean | |
| `error` | — | auto-panics on non-nil |
| `[]string` | array | |
| `[]byte` | string | cast |

Functions with non-bridgeable types (pointers, interfaces, channels, maps, structs, generics) are automatically excluded.

### Limitations

- Only exported package-level functions are bridged (no methods, structs, variables)
- Only top-level `.go` files are inspected (use `with` for sub-packages)
- Use `require` for wrapping Go libraries; use custom builds for stateful modules
