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
