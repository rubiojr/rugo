# Strings

Double-quoted strings support interpolation and escapes.

```ruby
name = "World"
puts "Hello, #{name}!"
puts "line1\nline2"
```

```text
Hello, World!
line1
line2
```

Single-quoted strings are raw:

```ruby
puts 'hello\nworld'
puts 'no #{interpolation}'
```

```text
hello\nworld
no #{interpolation}
```
