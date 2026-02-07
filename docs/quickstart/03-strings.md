# Strings

Double-quoted strings support interpolation with `#{}`:

```ruby
name = "World"
puts "Hello, #{name}!"
```

Expressions work inside interpolation:

```ruby
x = 10
puts "#{x} squared is #{x * x}"
```

Concatenation with `+`:

```ruby
greeting = "Hello" + ", " + "World!"
puts greeting
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
