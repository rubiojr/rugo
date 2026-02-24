---
name: rugo-rats
description: Expert in RATS (Rugo Automated Testing System), a BATS-like end-to-end testing framework for Rugo. Load when working on rats tests, the test runner, the test module, or writing _test.rugo files.
---

# RATS — Rugo Automated Testing System

RATS is a BATS-inspired end-to-end testing framework for Rugo. Tests live in `_test.rugo` files (or inline in regular `.rugo` files) and use the `rats` keyword with descriptive names.

## Test Syntax

```ruby
# test/myapp_test.rugo
use "test"

def setup()
  # runs before each test
end

def teardown()
  # runs after each test
end

rats "greets the user"
  result = test.run("./myapp greet World")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, World!")
end

rats "fails on missing arguments"
  result = test.run("./myapp greet")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "missing argument")
end

rats "can be skipped"
  test.skip("not ready yet")
end
```

## CLI

```bash
rugo rats                            # run all _test.rugo files in rats/ (or current dir)
rugo rats test/myapp_test.rugo         # run specific file
rugo rats myapp.rugo                   # run inline tests in a regular .rugo file
rugo rats --filter "greet"           # filter by test name
rugo rats -j 4                       # run with 4 parallel workers
rugo rats -j 1                       # run sequentially
rugo rats --tap                      # raw TAP output
rugo rats --recap                    # Print all failures with details at the end
rugo rats --timing                   # Per test and total elapsed time
```

## Output

```
$ rugo rats
 ✓ greets the user
 ✓ fails on missing arguments
 ✓ lists files
 - can be skipped (skipped: not ready yet)

4 tests, 3 passed, 0 failed, 1 skipped
```

TAP mode:
```
1..4
ok 1 greets the user
ok 2 fails on missing arguments
ok 3 lists files
ok 4 can be skipped # SKIP not ready yet
```

## `test` Stdlib Module

### `test.run(cmd)` — Run a command and capture results

Returns a hash with:
- `"status"` — exit code (integer)
- `"output"` — combined stdout+stderr (string)
- `"lines"` — output split by newlines (array)

```ruby
result = test.run("echo hello")
# result["status"]  → 0
# result["output"]  → "hello"
# result["lines"]   → ["hello"]
```

### Assertions

| Function | Description |
|----------|-------------|
| `test.assert_eq(actual, expected)` | Equal (`==`) |
| `test.assert_neq(actual, expected)` | Not equal (`!=`) |
| `test.assert_true(val)` | Truthy |
| `test.assert_false(val)` | Falsy |
| `test.assert_contains(str, substr)` | String contains substring |
| `test.assert_nil(val)` | Value is nil |
| `test.fail(msg)` | Explicitly fail the test |

### Flow control

| Function | Description |
|----------|-------------|
| `test.skip(reason)` | Skip the current test |

## How the Test Runner Works

The `rugo rats` command:

1. Discovers `_test.rugo` files (in `rats/` by default, or specified paths)
2. For each file:
   a. Parse and find all `rats "name" ... end` blocks
   b. Find `setup`/`teardown` if defined
   c. Generate a Go program that:
      - Defines each test as a function
      - Wraps each test in `defer recover()` to catch assertion panics
      - Calls `setup()` → test → `teardown()` for each
      - Outputs TAP format results
3. Compile and run the generated program
4. Parse output and display results

Assertions use `panic()` to abort the test on failure — Go's `recover()` catches them cleanly.

## Test Helpers

RATS supports shared helper files via a `helpers/` directory next to the test file. Any `.rugo` files in `helpers/` are automatically `require`d into the test file before parsing, so functions and constants defined there are available to all tests without explicit `require` statements.

```
my_tests/
  helpers/
    web_utils.rugo      # defines start_server(), etc.
    fixtures.rugo        # defines test data
  feature_test.rugo      # can call start_server() directly
```

The compiler generates `require "helpers/web_utils" as "web_utils"` (and so on) for each `.rugo` file in the directory. Helpers are only loaded in test mode (`rugo rats`), not during normal `rugo run`.

## Inline Tests

Tests can be embedded directly in regular `.rugo` files alongside normal code.
When run with `rugo run`, the `rats` blocks are silently ignored. When run
with `rugo rats`, they execute as tests.

```ruby
# math.rugo
use "test"

def add(a, b)
  return a + b
end

puts add(2, 3)

# Inline tests — ignored by `rugo run`, executed by `rugo rats`
rats "add returns the sum"
  test.assert_eq(add(1, 2), 3)
  test.assert_eq(add(-1, 1), 0)
end
```

```
$ rugo run math.rugo
5

$ rugo rats math.rugo
 ✓ add returns the sum
```

When scanning a directory, `rugo rats` discovers both `_test.rugo` files and
regular `.rugo` files containing `rats` blocks. Directories named `fixtures`
are skipped during discovery.

## Regression Test Suite

The `rats/` directory contains the project's regression test suite.

Fixtures live in `rats/fixtures/` (`.rugo` files for scripts, `_test.rugo` files for test fixtures).

## Do's and Don'ts When Writing RATS Tests

### DO: Inline code inside `rats` blocks (preferred)

Write assertions directly inside the `rats` block. This is the cleanest, most readable pattern and should be the default choice.

```ruby
use "test"
use "re"

rats "variables and arithmetic"
  x = 10
  y = 20
  test.assert_eq(x + y, 30)
end

rats "re.find returns first match"
  test.assert_eq(re.find("\\d+", "abc123def456"), "123")
end

rats "struct constructor"
  p = Point(10, 20)
  test.assert_eq(p.x, 10)
  test.assert_eq(p.y, 20)
end
```

### DO: Define functions and structs outside `rats` blocks

Top-level `def` functions, `struct` definitions, `use` imports, and `require` imports are visible to all `rats` blocks in the file. Use this to share helpers across tests.

```ruby
use "test"

struct Point
  x
  y
end

def Point.sum()
  return self.x + self.y
end

def get_name(obj)
  return obj.name
end

rats "method call"
  p = Point(3, 7)
  test.assert_eq(p.sum(), 10)
end

rats "helper function"
  h = {"name" => "Alice"}
  test.assert_eq(get_name(h), "Alice")
end
```

### DO: Use lambdas inline inside `rats` blocks

Lambda definitions work directly inside `rats` blocks.

```ruby
use "test"

rats "type_of lambda"
  f = fn(x) x + 1 end
  test.assert_eq(type_of(f), "Lambda")
end

rats "lambda as callback"
  nums = [1, 2, 3]
  doubled = map(nums, fn(n) n * 2 end)
  test.assert_eq(doubled, [2, 4, 6])
end
```

### DO: Use `eval.run()` with heredocs when inline is not possible

When you need to test error conditions, exit codes, compile errors, pipe chains, or features that require output capture (`puts`), use the `eval` module with heredocs. This compiles and runs the snippet in a subprocess.

```ruby
use "test"
use "eval"

rats "division by zero is a runtime error"
  source = <<~RUGO
    x = 5 / 0
  RUGO
  result = eval.run(source)
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "division by zero")
end

rats "pipe chain works"
  source = <<~RUGO
    use "str"
    echo "hello world" | str.upper | puts
  RUGO
  result = eval.run(source)
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "HELLO WORLD")
end
```

Use single-quoted heredoc delimiters (`<<~'RUGO'`) when the snippet contains string interpolation that should not be expanded by the outer test:

```ruby
rats "dot access on string is a runtime error"
  source = <<~'RUGO'
    x = "hello"
    puts(x.name)
  RUGO
  result = eval.run(source)
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot access .name on String")
end
```

### DON'T: Use fixtures and `test.run("rugo run ...")` unless unavoidable

Fixture-based tests that shell out to `rugo run rats/fixtures/some_file.rugo` are harder to read because the test logic lives in a separate file. **Avoid this pattern** unless you are testing CLI behavior, binary compilation, or process-level concerns that genuinely require a separate invocation.

Acceptable uses of `test.run()`:

```ruby
# OK — testing the CLI itself
rats "rugo --version works"
  result = test.run("rugo --version")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugo version")
end

# OK — testing binary compilation
rats "rugo build creates binary"
  result = test.run("rugo build -o #{test.tmpdir()}/hello examples/hello.rugo")
  test.assert_eq(result["status"], 0)
end
```

If you find yourself writing `test.run("rugo run rats/fixtures/...")`, stop and ask: can this be inlined in the `rats` block, or written with `eval.run()` and a heredoc? Almost always, yes.

### DON'T: Reference top-level variables or constants from inside `rats` blocks

Each `rats` block is an **isolated scope**. Top-level variables and constants are not visible inside `rats` blocks -- the compiler will emit an error.

```ruby
use "test"

port = 8080     # top-level variable
PORT = 9090     # top-level constant

rats "this will NOT compile"
  # WRONG — port and PORT are not in scope here
  test.assert_eq(port, 8080)   # compile error!
  test.assert_eq(PORT, 9090)   # compile error!
end
```

What IS shared across `rats` blocks:
- `def` functions
- `struct` definitions
- `use` module imports
- `require` imports

What is NOT shared:
- Variables (lowercase)
- Constants (UPPERCASE)
- Anything defined inside another `rats` block

### Pattern summary

| Pattern | When to use |
|---|---|
| **Inline** (preferred) | Value checks, module calls, structs, loops, conditions -- anything testable via direct assertions |
| **eval.run() + heredoc** (acceptable) | Error conditions, exit codes, pipe chains, compile errors, features needing `puts` output capture |
| **test.run() + fixtures** (avoid) | Only for CLI testing, binary compilation, or process-level behavior that cannot be expressed otherwise |

## Running RATS

**`make rats` is the canonical way to run the full RATS suite.** Always run it before finalizing any work.

When developing, run specific directories or files first for fast feedback, then finish with a single `make rats` to verify everything passes:

```bash
# Fast feedback during development — run specific dirs/files first
bin/rugo rats rats/core/                       # run core language tests
bin/rugo rats rats/stdlib/                     # run stdlib module tests
bin/rugo rats rats/gobridge/                   # run Go bridge tests
bin/rugo rats rats/tools/                      # run tool tests
bin/rugo rats rats/core/03_control_flow_test.rugo  # run a specific test file

# Then run the full suite before calling it done
make rats
```

## New Language Features Required (from design doc)

See `docs/rats.md` for the full design document including:
- Required language features (`rats` keyword, `str` module, test runner)
- Implementation order
- Generated Go code examples
- Feature comparison with BATS
