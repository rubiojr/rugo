# Chapter 1: Expressive Basics

Good Rugo code reads almost like prose. The language gives you several tools to
keep things clean and expressive — use them.

## String Interpolation Everywhere

Double-quoted strings support `#{}` interpolation. Prefer it over concatenation —
it's more readable and handles type conversion automatically.

```ruby
name = "Rugo"
version = 1
puts "Welcome to #{name} v#{version}!"
puts "#{2 + 2} is the answer to 2 + 2"
```

```
Welcome to Rugo v1!
4 is the answer to 2 + 2
```

Any expression works inside `#{}` — arithmetic, function calls, variable access.
The language calls `__to_s()` on the result, so you never need manual conversion
for display purposes.

## Drop the Parentheses

Rugo's preprocessor rewrites paren-free calls for you. When clarity isn't
sacrificed, drop the parens — it's the Rugo way.

```ruby
# Parentheses are optional for function calls
puts "hello, world"

def greet(name)
  puts "Hello, #{name}!"
end

greet "developer"
```

```
hello, world
Hello, developer!
```

Use parentheses when nesting calls or when the expression is complex. Drop them
when the intent is obvious. `puts "hello"` reads better than `puts("hello")` —
that's the whole point.

## Constants Are Uppercase

Any identifier starting with an uppercase letter is a constant — it can only be
assigned once. Use them for configuration values, magic numbers, and anything
that shouldn't change.

```ruby
PI = 3.14159
MAX_RETRIES = 3
AppName = "MyTool"

puts "#{AppName} uses PI=#{PI}, max retries: #{MAX_RETRIES}"
```

```
MyTool uses PI=3.14159, max retries: 3
```

Constants work inside functions too. Hash bindings are constant (you can't
reassign the variable) but the hash contents can still be mutated — just like
a `const` object in JavaScript.

## Compound Assignment

Use `+=`, `-=`, `*=`, `/=`, and `%=` to modify variables in place. They work
with numbers and strings alike.

```ruby
score = 0
score += 10
score += 25
score -= 5
puts "Final score: #{score}"

msg = "Hello"
msg += ", World!"
puts msg
```

```
Final score: 30
Hello, World!
```

## Raw Strings for Literal Content

Single-quoted strings are raw — no escape processing, no interpolation. Reach
for them when you need literal backslashes or want to preserve content exactly.

```ruby
path = 'C:\Users\name\Documents'
puts path

pattern = 'hello\nworld'
puts pattern

name = "Rugo"
puts 'no #{interpolation} here'
```

```
C:\Users\name\Documents
hello\nworld
no #{interpolation} here
```

Use double quotes when you need interpolation or escapes. Use single quotes
when you need the text exactly as written — regex patterns, Windows paths,
template strings.

## Heredocs for Multiline Text

When you need multiple lines, heredocs keep your code clean. The squiggly form
(`<<~`) strips common leading whitespace so your strings stay indented with
your code.

```ruby
name = "World"
greeting = <<~TEXT
  Hello, #{name}!
  Welcome to Rugo.
TEXT
puts greeting
```

```
Hello, World!
Welcome to Rugo.
```

There are four heredoc flavors:

| Syntax | Interpolation | Indent-stripping |
|--------|--------------|-----------------|
| `<<DELIM` | ✓ | ✗ |
| `<<~DELIM` | ✓ | ✓ |
| `<<'DELIM'` | ✗ | ✗ |
| `<<~'DELIM'` | ✗ | ✓ |

Reach for `<<~DELIM` most of the time — it gives you both interpolation and
clean indentation.
