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

## Constants

Names starting with an uppercase letter are constants — they can only be assigned once:

```ruby
PI = 3.14
MAX_RETRIES = 5
AppName = "MyApp"

PI = 99   # compile error: cannot reassign constant PI
```

Lowercase names remain freely reassignable. This follows Ruby convention.

## Scoping

Different blocks have different scoping rules:

### Functions have their own scope

Variables inside a function are local — they can't see top-level variables, and they don't leak out:

```ruby
name = "Rugo"

def greet()
  name = "World"     # this is a separate variable
  return name
end

puts greet()   # World
puts name      # Rugo
```

Function parameters are also local:

```ruby
def add(a, b)
  return a + b
end

puts add(1, 2)  # 3
# puts a         # compile error: undefined: a
```

### `if` blocks share the parent scope

Variables created or modified inside `if`/`elsif`/`else` are visible after the block:

```ruby
if true
  msg = "hello"
end
puts msg  # hello

x = "before"
if true
  x = "after"
end
puts x  # after
```

### Loops create their own scope

`while` and `for` loops can read and modify outer variables, but new variables defined inside the loop stay local:

```ruby
total = 0
for x in [1, 2, 3]
  total += x
end
puts total  # 6
# puts x    # compile error: undefined: x

count = 0
while count < 3
  last = count
  count += 1
end
puts count  # 3
# puts last # compile error: undefined: last
```

### Lambdas capture the outer scope

Unlike functions, lambdas can see variables from the surrounding scope:

```ruby
prefix = "Hello"
greet = fn(name) prefix + ", " + name end
puts greet("Rugo")  # Hello, Rugo
```

But variables defined inside a lambda don't leak out:

```ruby
maker = fn()
  secret = "hidden"
  return secret
end
puts maker()  # hidden
# puts secret # compile error: undefined: secret
```

### Constants are scoped per function

A constant in a function is independent from one with the same name at the top level:

```ruby
PI = 3.14

def show()
  PI = 99
  puts PI
end

show()    # 99
puts PI   # 3.14
```

---
Next: [Strings](03-strings.md)
