# conv

Type conversion utilities.

```ruby
use "conv"
```

## to_s

Converts any value to a string.

```ruby
s = conv.to_s(42)       # "42"
s = conv.to_s(3.14)     # "3.14"
s = conv.to_s(true)     # "true"
```

## to_i

Converts a value to an integer.

```ruby
n = conv.to_i("42")     # 42
n = conv.to_i(3.9)      # 3
n = conv.to_i(true)     # 1
n = conv.to_i(false)    # 0
```

Panics if the string is not a valid integer.

## to_f

Converts a value to a float.

```ruby
f = conv.to_f("3.14")   # 3.14
f = conv.to_f(42)       # 42.0
```

Panics if the string is not a valid number.

## to_bool

Converts a value to a boolean.

```ruby
conv.to_bool(1)        # true
conv.to_bool(0)        # false
conv.to_bool("")       # false
conv.to_bool("hello")  # true
conv.to_bool("false")  # false
conv.to_bool(nil)      # false
```

Rules: `false`, `0`, `0.0`, `""`, `"false"`, and `nil` are falsy. Everything else is truthy.

## parse_int

Parses a string as an integer with a given base.

```ruby
conv.parse_int("ff", 16)       # 255
conv.parse_int("1010", 2)      # 10
conv.parse_int("77", 8)        # 63
conv.parse_int("42", 10)       # 42
```

Panics if the string is not valid for the given base.
