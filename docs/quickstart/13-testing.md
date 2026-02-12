# Testing with RATS

Rugo includes **RATS** (Rugo Automated Testing System), a built-in test
framework inspired by BATS. Tests live in `_test.rugo` files and use the `test` module.

## Writing Your First Test

Create a file called `test/greet_test.rugo`:

```ruby
use "test"

rats "prints hello"
  result = test.run("rugo run greet.rugo")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end
```

## Running Tests

```bash
rugo rats                       # run all _test.rugo files in rats/ (or current dir)
rugo rats test/greet_test.rugo         # run a specific file
rugo rats --filter "hello"      # filter by test name
rugo rats --timing              # show per-test and total elapsed time
```

Output looks like:

```
 ✓ prints hello

1 tests, 1 passed, 0 failed
```

## Capturing Command Output

`test.run(cmd)` executes a shell command and returns a hash with:

- `"status"` — exit code (integer)
- `"output"` — combined stdout+stderr (string)
- `"lines"` — output split by newlines (array)

```ruby
use "test"

rats "captures output lines"
  result = test.run("printf 'a\nb\nc'")
  test.assert_eq(result["status"], 0)
  test.assert_eq(len(result["lines"]), 3)
  test.assert_eq(result["lines"][0], "a")
end
```

## Assertions

| Function | Description |
|----------|-------------|
| `test.assert_eq(a, b)` | Equal (`==`) |
| `test.assert_neq(a, b)` | Not equal (`!=`) |
| `test.assert_true(val)` | Truthy |
| `test.assert_false(val)` | Falsy |
| `test.assert_contains(s, sub)` | String contains substring |
| `test.assert_nil(val)` | Value is nil |
| `test.fail(msg)` | Explicitly fail |

## Skipping Tests

```ruby
rats "not ready yet"
  test.skip("pending feature")
  test.fail("should not reach here")
end
```

Skipped tests show in output:

```
 - not ready yet (skipped: pending feature)
```

## Testing a Built Binary

You can build a Rugo script and test the resulting binary:

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

---
That's it! RATS gives you a simple, built-in way to test your Rugo scripts.
See the [RATS design doc](../rats.md) for the full specification and the
`test` module's complete API.

## Setup and Teardown

RATS supports four hook functions for test lifecycle management:

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
  # Create shared resources once for all tests
  os.exec("mkdir -p /tmp/myapp_test")
end

def teardown_file()
  # Clean up shared resources after all tests
  os.exec("rm -rf /tmp/myapp_test")
end

def setup()
  # Reset state before each test
  test.write_file(test.tmpdir() + "/input.txt", "default")
end

def teardown()
  # Clean up per-test state (tmpdir is auto-cleaned)
end

rats "uses shared resource"
  test.write_file("/tmp/myapp_test/data.txt", "hello")
  result = test.run("cat /tmp/myapp_test/data.txt")
  test.assert_eq(result["output"], "hello")
end
```

The execution order is:

```
setup_file()
  for each test:
    create tmpdir
    setup()
    run test
    teardown()
    clean tmpdir
teardown_file()
```

`teardown_file()` always runs, even if tests fail.

## Inline Tests

You can embed `rats` blocks directly in regular `.rugo` files. When you run the
file normally with `rugo run`, the test blocks are silently ignored. When you
run the file with `rugo rats`, only the test blocks execute.

```ruby
# greet.rugo
use "test"

def greet(name)
  return "Hello, " + name + "!"
end

puts greet("World")

# These tests are ignored by `rugo run`, executed by `rugo rats`
rats "greet formats a greeting"
  test.assert_eq(greet("Rugo"), "Hello, Rugo!")
  test.assert_contains(greet("World"), "World")
end
```

```bash
rugo run greet.rugo       # prints "Hello, World!" — tests ignored
rugo rats greet.rugo      # runs the inline tests
```

When scanning a directory, `rugo rats` automatically discovers both `_test.rugo`
files and regular `.rugo` files that contain `rats` blocks (directories named
`fixtures` are skipped).
