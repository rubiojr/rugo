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

## Heredoc Strings

Heredocs are multiline string literals. Use `<<DELIM` to start a heredoc and `DELIM` on its own line to close it. Delimiters must be uppercase (`[A-Z_][A-Z0-9_]*`).

### Interpolating heredoc

Works like double-quoted strings — `#{...}` is evaluated:

```ruby
name = "World"
html = <<HTML
<h1>Hello #{name}</h1>
<p>Welcome!</p>
HTML
puts html
```

### Squiggly heredoc (`<<~`)

Strips common leading whitespace so you can indent the body with your code:

```ruby
name = "World"
page = <<~HTML
  <h1>Hello #{name}</h1>
  <p>Welcome!</p>
HTML
puts page  # no leading spaces in output
```

### Raw heredoc (`<<'DELIM'`)

No interpolation — content is literal, like single-quoted strings:

```ruby
template = <<'CODE'
def #{method_name}
  puts "hello"
end
CODE
puts template  # #{method_name} printed literally
```

### Raw squiggly heredoc (`<<~'DELIM'`)

Combines indent stripping with literal content:

```ruby
config = <<~'YAML'
  name: myapp
  version: 1.0
YAML
```

The closing delimiter can be indented — leading whitespace is ignored when matching.

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

## String Comparison

Strings support all comparison operators with lexicographic ordering:

```ruby
if "apple" < "banana"
  puts "apple comes first"
end

if "hello" == "hello"
  puts "equal"
end
```

## String Module

Import `str` for string utilities:

```ruby
use "str"

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

puts str.join(parts, " | ")          # a | b | c
```

---
Next: [Arrays](04-arrays.md)
