# Variables

Variables are dynamically typed. No declarations needed.

```ruby
name = "Rugo"
age = 1
pi = 3.14
cool = true
nothing = nil
```

Reassignment works freely:

```ruby
x = 10
x = "now a string"
```

## Compound Assignment

```ruby
x = 10
x += 5   # 15
x -= 3   # 12
x *= 2   # 24
x /= 4   # 6
x %= 4   # 2
```

Works with strings too:

```ruby
msg = "Hello"
msg += ", World!"
puts msg
```

---
Next: [Strings](03-strings.md)
