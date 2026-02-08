# test

Testing and assertions for Rugo scripts.

```ruby
use "test"
```

## run

Executes a shell command and returns a hash with `"status"` (exit code), `"output"` (combined stdout/stderr), and `"lines"` (output split into an array).

```ruby
result = test.run("echo hello")
test.assert_eq(result["status"], 0)
test.assert_eq(result["output"], "hello")
```

## assert_eq / assert_neq

Assert two values are equal or not equal.

```ruby
test.assert_eq(1 + 1, 2)
test.assert_neq("a", "b")
```

## assert_true / assert_false

Assert a value is truthy or falsy.

```ruby
test.assert_true(len("hi") > 0)
test.assert_false(nil)
```

## assert_contains

Assert a string contains a substring.

```ruby
test.assert_contains("hello world", "world")
```

## assert_nil

Assert a value is `nil`.

```ruby
result = try `nonexistent_cmd`
test.assert_nil(result)
```

## fail

Immediately fail the test with a message.

```ruby
test.fail("not implemented yet")
```

## skip

Skip the current test with a reason.

```ruby
test.skip("requires network")
```
