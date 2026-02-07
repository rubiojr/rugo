# Strings

Double-quoted strings support escape sequences and interpolation with `#{}`:

```ruby
name = "World"
puts "Hello, #{name}!"
```

Expressions work inside interpolation:

```ruby
x = 10
puts "#{x} squared is #{x * x}"
```

## Raw Strings

Single-quoted strings are raw — no escape processing and no interpolation:

```ruby
puts 'hello\nworld'       # prints: hello\nworld (literal, no newline)
puts '\x1b[32mgreen'      # prints: \x1b[32mgreen (literal, no ANSI)
puts 'no #{interpolation}' # prints: no #{interpolation}
```

Only `\\` (literal backslash) and `\'` (literal single quote) are recognized:

```ruby
puts 'it\'s raw'          # prints: it's raw
puts 'back\\slash'        # prints: back\slash
```

Raw strings are useful for regex patterns, Windows paths, and test assertions where you need exact literal text.

## Concatenation

Concatenation with `+`:

```ruby
greeting = "Hello" + ", " + "World!"
puts greeting
```

Raw and double-quoted strings can be concatenated:

```ruby
puts 'raw\n' + "escaped\n"  # raw\nescaped<newline>
```

## String Module

Import `str` for string utilities:

```ruby
import "str"

puts str.upper("hello")              # HELLO
puts str.lower("HELLO")              # hello
puts str.trim("  hello  ")           # hello
puts str.contains("hello", "ell")    # true
puts str.starts_with("hello", "he")  # true
puts str.ends_with("hello", "lo")    # true
puts str.replace("hello", "l", "r")  # herro
puts str.index("hello", "ll")        # 2

parts = str.split("a,b,c", ",")
puts parts
```

---
Next: [Arrays](04-arrays.md)
