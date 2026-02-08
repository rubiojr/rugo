---
name: rugo-rats
description: Expert in RATS (Rugo Automated Testing System), a BATS-like end-to-end testing framework for Rugo. Load when working on rats tests, the test runner, the test module, or writing _test.rg files.
---

# RATS — Rugo Automated Testing System

RATS is a BATS-inspired end-to-end testing framework for Rugo. Tests live in `_test.rg` files (or inline in regular `.rg` files) and use the `rats` keyword with descriptive names.

## Test Syntax

```ruby
# test/myapp_test.rg
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

### Grammar

```
TestDef = "rats" str_lit Body "end" .
```

The `rats` keyword follows the same pattern as `FuncDef` but with a string description instead of an ident name and no parameters.

## CLI

```bash
rugo rats                            # run all _test.rg files in rats/ (or current dir)
rugo rats test/myapp_test.rg         # run specific file
rugo rats myapp.rg                   # run inline tests in a regular .rg file
rugo rats --filter "greet"           # filter by test name
rugo rats -j 4                       # run with 4 parallel workers
rugo rats -j 1                       # run sequentially
rugo rats --tap                      # raw TAP output
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

1. Discovers `_test.rg` files (in `rats/` by default, or specified paths)
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

## Regression Test Suite

The `rats/` directory contains the project's regression test suite:

| File | Coverage |
|------|----------|
| `rats/13_spawn_test.rg` | 21 tests: block, one-liner, fan-out, try/or, .done, .wait, functions, empty body, codegen gating, native binary, 5 negative tests |
| `rats/14_parallel_test.rg` | 11 tests: ordered results, shell commands, single expr, nested, try/or, empty body, import gating, native binary, 2 negative tests |
| `rats/28_bench_test.rg` | 4 tests: basic bench, multi bench, bench with functions, bench keyword in emit |
| `rats/gobridge/` | 60 tests across 7 files covering all 8 Go bridge packages plus edge cases and aliasing |

Fixtures live in `rats/fixtures/` (`.rg` files for scripts, `_test.rg` files for test fixtures).

## Running RATS

```bash
rugo rats rats/                           # run all regression tests
rugo rats rats/03_control_flow_test.rg    # run a specific test file
```

## New Language Features Required (from design doc)

See `docs/rats.md` for the full design document including:
- Required language features (`rats` keyword, `str` module, test runner)
- Implementation order
- Generated Go code examples
- Feature comparison with BATS
