# Functions

## Define and Call

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

greet("World")
```

## No-Argument Functions

Functions with no parameters can omit the parentheses in the definition:

```ruby
def say_hello
  puts "Hello!"
end

say_hello
```

Both `def say_hello` and `def say_hello()` are valid — the `()` is optional when there are no parameters.

## Return Values

The last expression in a function body is implicitly returned:

```ruby
def add(a, b)
  a + b
end

puts add(2, 3)   # 5
```

Use explicit `return` for early exits:

```ruby
def classify(x)
  if x > 10
    return "big"
  end
  "small"
end

puts classify(5)    # small
puts classify(20)   # big
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
  n * factorial(n - 1)
end

puts factorial(5)   # 120
```

---
Next: [Lambdas](08b-lambdas.md)
