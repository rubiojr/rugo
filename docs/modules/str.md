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

## join

Joins an array into a string with the given separator.

```ruby
parts = ["a=1", "b=2", "c=3"]
str.join(parts, "&")   # "a=1&b=2&c=3"
str.join([1, 2, 3], "-")   # "1-2-3"
```

## rune_count

Returns the number of Unicode characters (runes) in a string. Unlike `len`, which returns byte count, `rune_count` counts visible characters.

```ruby
str.rune_count("hello")   # 5
str.rune_count("café")    # 4 (len returns 5)
str.rune_count("日本語")   # 3
```

## count

Counts non-overlapping occurrences of a substring.

```ruby
str.count("hello world", "l")   # 3
str.count("aaa", "aa")          # 1
```

## repeat

Repeats a string n times.

```ruby
str.repeat("ab", 3)   # "ababab"
str.repeat("x", 0)    # ""
```

## reverse

Reverses a string by Unicode characters.

```ruby
str.reverse("hello")   # "olleh"
str.reverse("café")    # "éfac"
```

## chars

Splits a string into an array of individual characters.

```ruby
str.chars("hi")     # ["h", "i"]
str.chars("café")   # ["c", "a", "f", "é"]
```

## fields

Splits a string by whitespace into an array of words.

```ruby
str.fields("  a  b  c  ")   # ["a", "b", "c"]
```

## trim_prefix / trim_suffix

Removes a prefix or suffix from a string if present.

```ruby
str.trim_prefix("hello", "he")   # "llo"
str.trim_suffix("hello", "lo")   # "hel"
```

## pad_left

Left-pads a string to a given width. Optional third argument is the pad character (default: space).

```ruby
str.pad_left("hi", 5)        # "   hi"
str.pad_left("hi", 5, "0")   # "000hi"
```

## pad_right

Right-pads a string to a given width. Optional third argument is the pad character (default: space).

```ruby
str.pad_right("hi", 5)        # "hi   "
str.pad_right("hi", 5, ".")   # "hi..."
```

## center

Centers a string within a given width. Optional third argument is the pad character (default: space).

```ruby
str.center("hi", 6)        # "  hi  "
str.center("hi", 6, "-")   # "--hi--"
```

## each_line

Splits a string into an array of lines.

```ruby
lines = str.each_line("a\nb\nc")   # ["a", "b", "c"]
```

## last_index

Returns the index of the last occurrence of a substring, or `-1` if not found.

```ruby
str.last_index("hello", "l")     # 3
str.last_index("hello", "xyz")   # -1
```

## slice

Extracts a substring by rune start and end indices. Supports negative indices (counted from end).

```ruby
str.slice("hello", 1, 3)     # "el"
str.slice("hello", -3, -1)   # "ll"
str.slice("café", 3, 4)      # "é"
```

## empty

Returns `true` if the string is empty.

```ruby
str.empty("")    # true
str.empty("x")   # false
```
