# RATS — Rugo Automated Testing System

Research document for a BATS-like end-to-end testing framework for Rugo.

## BATS Core Concepts

BATS (Bash Automated Testing System) works like this:

```bash
# test/myapp.bats

setup() {
  # runs before each test
}

teardown() {
  # runs after each test
}

@test "greets the user" {
  run ./myapp greet World
  [ "$status" -eq 0 ]
  [ "$output" = "Hello, World!" ]
}

@test "fails on missing args" {
  run -1 ./myapp greet
  [[ "$output" =~ "missing argument" ]]
}
```

Key features:
- `@test "name" { ... }` — test blocks with descriptive names
- `run cmd` — captures exit status + output into `$status`, `$output`, `$lines`
- `setup`/`teardown` — per-test hooks
- `setup_file`/`teardown_file` — per-file hooks
- `skip "reason"` — skip tests
- `load helper` — share code across test files
- TAP output format
- Parallel execution via `--jobs`

## Proposed Rugo Design

### Test file syntax (`_test.rg` files)

Tests can live in dedicated `_test.rg` files or inline in regular `.rg` files.

```ruby
# test/myapp_test.rg
use "test"
use "os"

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

rats "lists files"
  result = test.run("ls /tmp")
  test.assert_eq(result["status"], 0)
  test.assert_true(len(result["lines"]) > 0)
end

rats "can be skipped"
  test.skip("not ready yet")
end
```

### CLI

```bash
rugo rats                            # run all test files in rats/ (or current dir)
rugo rats test/myapp_test.rg         # run specific file
rugo rats myapp.rg                   # run inline tests in a regular .rg file
rugo rats --filter "greet"           # filter by test name
rugo rats -j 4                       # run with 4 parallel workers
rugo rats -j 1                       # run sequentially
rugo rats --tap                      # raw TAP output
```

### Output

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

The `rugo rats` command would:

1. Discover `_test.rg` files (in `test/` by default, or specified paths)
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

### Generated Go (simplified)

For a test like:
```ruby
rats "greets the user"
  result = test.run("./myapp greet World")
  test.assert_eq(result["status"], 0)
end
```

The runner generates:
```go
func rugotest_greets_the_user() (passed bool, skipMsg string) {
    defer func() {
        if r := recover(); r != nil {
            if skip, ok := r.(rugoTestSkip); ok {
                skipMsg = string(skip)
                return
            }
            passed = false
            fmt.Fprintf(os.Stderr, "  FAIL: %v\n", r)
        }
    }()
    // ... test body ...
    passed = true
    return
}
```

Assertions use `panic()` to abort the test on failure — Go's `recover()` catches them cleanly.

## New Language Features Required

### 1. `rats "name" ... end` block (Required)

New grammar production:
```
TestDef = "rats" str_lit Body "end" .
```

This is like `def` but with a string description instead of an ident name, and no parameters. The `rats` keyword must be added to the grammar and the preprocessor keyword list.

**Effort: Small** — follows the same pattern as `FuncDef`.

### 2. String utility functions (Required)

Assertions like `assert_contains` need string operations. Two options:

**Option A: `str` stdlib module**
```ruby
use "str"
str.contains("hello world", "world")  # true
str.split("a,b,c", ",")               # ["a", "b", "c"]
str.trim("  hello  ")                  # "hello"
```

**Option B: Add to `conv` or as global builtins**
```ruby
contains("hello world", "world")
split("a,b,c", ",")
```

**Recommendation: Option A** — keeps the language clean, consistent with import system.

Functions needed:
- `str.contains(s, substr)` → bool
- `str.split(s, sep)` → array
- `str.trim(s)` → string
- `str.starts_with(s, prefix)` → bool
- `str.ends_with(s, suffix)` → bool
- `str.replace(s, old, new)` → string

**Effort: Small** — all are one-liners wrapping Go `strings` package.

### 3. Test runner (`rugo rats` command) (Required)

A new subcommand in `main.go` that:
- Scans for `_test.rg` files
- Parses them to find test blocks
- Generates a special Go program with test harness
- Compiles and runs it

**Effort: Medium** — similar to the existing `run`/`build` pipeline but with test harness generation.

### 4. `test.run()` returning a hash (Required)

The `test.run(cmd)` function needs to:
- Execute a command
- Capture stdout+stderr
- Capture exit code
- Return a hash: `{"status" => code, "output" => str, "lines" => arr}`

This works today with Rugo's existing hash and array types. The function is a stdlib runtime function that returns `map[interface{}]interface{}`.

**Effort: Small** — straightforward Go implementation.

### 5. Error recovery via `recover()` (Required, but free)

Go's `recover()` in the generated test harness catches assertion panics. Assertions call `panic("assert_eq failed: got X, want Y")`. The harness catches this, marks the test as failed, and continues to the next test.

**Effort: None** — this is purely in the generated Go code, no language change needed.

## Features NOT Required (vs BATS)

These BATS features can be deferred or aren't needed:

| BATS Feature | RATS Status | Reason |
|---|---|---|
| `setup_file`/`teardown_file` | Defer | `setup`/`teardown` per-test is sufficient initially |
| `--jobs` parallel | ✅ Done | `rugo rats -j N`, defaults to NumCPU |
| `--filter-tags` | Defer | `--filter` regex is enough |
| `load` helper | Already have `require` | `require "test_helper"` works |
| `bats_pipe` | Not needed | `test.run("cmd1 \| cmd2")` works since it runs via `sh -c` |
| `$BATS_TEST_*` variables | Defer | Nice-to-have, not essential |

## Implementation Order

1. **`str` stdlib module** — needed by assertions, useful generally
2. **`rats "name" ... end` syntax** — grammar + walker + codegen
3. **`test` stdlib module** — `run()`, assertions, `skip()`, `fail()`
4. **`rugo rats` command** — test runner with TAP output
5. **Pretty output formatter** — ✓/✗ display with colors

## Example: Testing a Rugo Script

```ruby
# greet.rg
use "os"

def greet(name)
  if name == ""
    puts("Error: name required")
    os.exit(1)
  end
  puts("Hello, " + name + "!")
end

greet("World")
```

```ruby
# test/greet_test.rg
use "test"

rats "outputs greeting"
  result = test.run("rugo run greet.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello, World!")
end

rats "greet binary works"
  test.run("rugo build greet.rg -o /tmp/greet")
  result = test.run("/tmp/greet")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, World!")
end
```

```
$ rugo rats test/greet_test.rg
 ✓ outputs greeting
 ✓ greet binary works

2 tests, 2 passed, 0 failed
```

## Inline Tests

Tests can be embedded directly in regular `.rg` files alongside normal code.
When run with `rugo run`, the `rats` blocks are silently ignored. When run
with `rugo rats`, they execute as tests.

```ruby
# math.rg
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
$ rugo run math.rg
5

$ rugo rats math.rg
 ✓ add returns the sum
```

When scanning a directory, `rugo rats` discovers both `_test.rg` files and
regular `.rg` files containing `rats` blocks. Directories named `fixtures`
are skipped during discovery.
