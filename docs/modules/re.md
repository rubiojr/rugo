# re

Regular expressions using Go's `regexp` package.

```ruby
use "re"
```

All functions take the pattern as the first argument. Invalid patterns panic (use `try/or` to handle).

## Functions

### `re.test(pattern, s)`

Returns `true` if the pattern matches anywhere in the string.

```ruby
re.test("^\\d+$", "42")      # true
re.test("^\\d+$", "abc")     # false
```

### `re.find(pattern, s)`

Returns the first match, or `nil` if no match.

```ruby
re.find("\\d+", "abc123def")   # "123"
re.find("\\d+", "no numbers")  # nil
```

### `re.find_all(pattern, s)`

Returns all matches as an array.

```ruby
re.find_all("\\d+", "a1b2c3")  # ["1", "2", "3"]
```

### `re.replace(pattern, s, replacement)`

Replaces the first match.

```ruby
re.replace("\\d+", "a1b2c3", "X")  # "aXb2c3"
```

### `re.replace_all(pattern, s, replacement)`

Replaces all matches. Supports backreferences (`$1`, `$2`).

```ruby
re.replace_all("\\d+", "a1b2c3", "X")                     # "aXbXcX"
re.replace_all("(\\w+)@(\\w+)", "foo@bar", "$1 at $2")     # "foo at bar"
```

### `re.split(pattern, s)`

Splits the string by the pattern. Returns an array.

```ruby
re.split("\\s+", "hello   world")   # ["hello", "world"]
re.split(",\\s*", "a, b, c")        # ["a", "b", "c"]
```

### `re.match(pattern, s)`

Returns a hash with `"match"` (full match string) and `"groups"` (array of capture groups), or `nil` if no match.

```ruby
m = re.match("(\\w+)@(\\w+)", "foo@bar.com")
m["match"]      # "foo@bar"
m["groups"][0]   # "foo"
m["groups"][1]   # "bar"
```

## Error Handling

Invalid patterns panic with a descriptive message. Use `try/or` to handle:

```ruby
result = try re.test("[invalid", "test") or "bad pattern"
```
