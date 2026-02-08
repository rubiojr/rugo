# str

String utilities.

```ruby
use "str"
```

## upper / lower

```ruby
str.upper("hello")   # HELLO
str.lower("HELLO")   # hello
```

## trim

Removes leading and trailing whitespace.

```ruby
str.trim("  hello  ")   # hello
```

## contains

Returns `true` if the string contains the substring.

```ruby
str.contains("hello world", "world")   # true
```

## starts_with / ends_with

```ruby
str.starts_with("hello", "he")   # true
str.ends_with("hello", "lo")     # true
```

## replace

Replaces all occurrences of a substring.

```ruby
str.replace("hello", "l", "r")   # herro
```

## split

Splits a string by a separator. Returns an array.

```ruby
parts = str.split("a,b,c", ",")
puts parts   # ["a", "b", "c"]
```

## index

Returns the index of the first occurrence of a substring, or `-1` if not found.

```ruby
str.index("hello", "ll")    # 2
str.index("hello", "xyz")   # -1
```
