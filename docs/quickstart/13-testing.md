# Testing with RATS

Rugo includes **RATS** (Rugo Automated Testing System), a built-in test
framework inspired by BATS. Tests live in `_test.rg` files and use the `test` module.

## Writing Your First Test

Create a file called `test/greet_test.rg`:

```ruby
import "test"

rats "prints hello"
  result = test.run("rugo run greet.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end
```

## Running Tests

```bash
rugo rats                       # run all _test.rg files in test/
rugo rats test/greet_test.rg         # run a specific file
rugo rats --filter "hello"      # filter by test name
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
import "test"

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
import "test"

rats "binary works"
  test.run("rugo build greet.rg -o /tmp/greet")
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
