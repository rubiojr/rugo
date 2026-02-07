# Functions

## Define and Call

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

greet("World")
```

## Return Values

```ruby
def add(a, b)
  return a + b
end

puts add(2, 3)   # 5
```

## Parenthesis-Free Calls

When calling with arguments, parentheses are optional:

```ruby
puts "hello"
greet "World"
```

## Recursion

```ruby
def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

puts factorial(5)   # 120
```

---
Next: [Shell Commands](09-shell.md)
