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

### Grammar

```
TestDef = "rats" str_lit Body "end" .
```

The `rats` keyword follows the same pattern as `FuncDef` but with a string description instead of an ident name and no parameters.

## CLI

```bash
rugo rats                            # run all _test.rugo files in rats/ (or current dir)
rugo rats test/myapp_test.rugo         # run specific file
rugo rats myapp.rugo                   # run inline tests in a regular .rugo file
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

The `rats/` directory contains the project's regression test suite (54 test files, 600+ tests):

| File | Tests | Coverage area |
|------|-------|---------------|
| `rats/01_cli_test.rugo` | 9 | CLI basics |
| `rats/02_variables_test.rugo` | 8 | Variable assignment |
| `rats/03_control_flow_test.rugo` | 4 | if/else, loops |
| `rats/04_functions_test.rugo` | 9 | Function definitions |
| `rats/05_data_structures_test.rugo` | 5 | Arrays, hashes |
| `rats/06_modules_test.rugo` | 9 | Module system |
| `rats/07_require_shell_test.rugo` | 3 | require + shell |
| `rats/08_rats_self_test.rugo` | 10 | RATS self-tests |
| `rats/09_try_or_test.rugo` | 5 | try/or error handling |
| `rats/10_error_output_test.rugo` | 7 | Error output formatting |
| `rats/10_raise_test.rugo` | 5 | raise keyword |
| `rats/11_backticks_test.rugo` | 4 | Backtick shell exec |
| `rats/12_syntax_errors_test.rugo` | 21 | Syntax error reporting |
| `rats/13_spawn_test.rugo` | 29 | spawn concurrency |
| `rats/14_parallel_test.rugo` | 11 | parallel blocks |
| `rats/15_test_colors_test.rugo` | 8 | Color output |
| `rats/16_cli_module_test.rugo` | 17 | CLI module |
| `rats/17_color_module_test.rugo` | 7 | Color module |
| `rats/18_escape_sequences_test.rugo` | 9 | Escape sequences |
| `rats/19_json_module_test.rugo` | 10 | JSON module |
| `rats/20_custom_modules_test.rugo` | 4 | Custom modules |
| `rats/21_summary_test.rugo` | 2 | Test summary output |
| `rats/22_raw_strings_test.rugo` | 8 | Raw strings |
| `rats/23_comparisons_test.rugo` | 8 | Comparison operators |
| `rats/24_constants_test.rugo` | 8 | Constants |
| `rats/25_arg_count_test.rugo` | 6 | Argument count checks |
| `rats/26_negative_index_test.rugo` | 5 | Negative indexing |
| `rats/27_pipes_test.rugo` | 14 | Pipe operator |
| `rats/28_bench_test.rugo` | 4 | Benchmarks |
| `rats/29_module_edge_cases_test.rugo` | 8 | Module edge cases |
| `rats/30_heredoc_test.rugo` | 12 | Heredocs |
| `rats/31_structs_test.rugo` | 69 | Structs |
| `rats/32_hash_keys_test.rugo` | 12 | Hash keys |
| `rats/33_hash_colon_syntax_test.rugo` | 16 | Hash colon syntax |
| `rats/34_web_module_test.rugo` | 31 | Web module |
| `rats/35_web_middleware_test.rugo` | 16 | Web middleware |
| `rats/36_inline_tests_test.rugo` | 4 | Inline tests |
| `rats/37_error_ux_test.rugo` | 32 | Error UX |
| `rats/38_def_optional_parens_test.rugo` | 11 | Optional parens |
| `rats/39_type_inference_test.rugo` | 25 | Type inference |
| `rats/40_test_timeout_test.rugo` | 3 | Test timeouts |
| `rats/41_lambdas_test.rugo` | 19 | Lambdas |
| `rats/42_http_module_test.rugo` | 10 | HTTP module |
| `rats/43_require_typed_call_test.rugo` | 2 | Typed require calls |
| `rats/44_subdir_require_test.rugo` | 2 | Subdir require |
| `rats/45_require_scope_test.rugo` | 3 | Require scoping |
| `rats/46_type_of_test.rugo` | 19 | type_of() |
| `rats/47_if_scope_test.rugo` | 5 | If block scoping |
| `rats/48_bare_variable_test.rugo` | 11 | Bare variable errors |
| `rats/49_remote_require_test.rugo` | 5 | Remote require |
| `rats/50_setup_file_test.rugo` | 2 | setup_file/teardown_file |
| `rats/51_setup_teardown_combined_test.rugo` | 3 | Combined hooks |
| `rats/fmt_test.rugo` | 9 | fmt module |
| `rats/re_test.rugo` | 16 | Regex module |
| `rats/gobridge/` | 60 | 7 files covering Go bridge packages |

Fixtures live in `rats/fixtures/` (`.rugo` files for scripts, `_test.rugo` files for test fixtures).

## Running RATS

```bash
rugo rats rats/                           # run all regression tests
rugo rats rats/03_control_flow_test.rugo    # run a specific test file
```

## New Language Features Required (from design doc)

See `docs/rats.md` for the full design document including:
- Required language features (`rats` keyword, `str` module, test runner)
- Implementation order
- Generated Go code examples
- Feature comparison with BATS
