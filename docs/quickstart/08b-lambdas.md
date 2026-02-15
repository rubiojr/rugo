# Lambdas (First-Class Functions)

Rugo supports anonymous functions (lambdas) using `fn...end` syntax. Lambdas can be stored in variables, passed to functions, returned from functions, and stored in data structures.

## Basic Lambda

```ruby
double = fn(x) x * 2 end
puts double(5)   # 10
```

## Multi-Line Lambda

```ruby
classify = fn(x)
  if x > 0
    return "positive"
  end
  return "non-positive"
end

puts classify(1)    # positive
puts classify(-1)   # non-positive
```

## Passing Lambdas to Functions

```ruby
def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end

nums = my_map(fn(x) x * 2 end, [1, 2, 3])
puts nums   # [2, 4, 6]
```

## Closures

Functions returned from other functions capture their enclosing scope:

```ruby
def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
puts add5(10)   # 15
```

Closures capture by reference — changes to the outer variable are visible inside the lambda:

```ruby
x = 10
f = fn() x end
x = 20
puts f()   # 20
```

Closures can also mutate captured variables:

```ruby
def make_counter()
  count = 0
  inc = fn()
    count = count + 1
    return count
  end
  return inc
end

counter = make_counter()
puts counter()   # 1
puts counter()   # 2
```

## Lambdas in Data Structures

```ruby
ops = {
  "add" => fn(a, b) a + b end,
  "mul" => fn(a, b) a * b end
}
puts ops["add"](2, 3)   # 5

fns = [fn(x) x + 1 end, fn(x) x * 2 end]
puts fns[0](10)   # 11
```

## Calling Lambdas via Dot Access

Lambdas stored in hashes can be called using dot syntax, just like index access:

```ruby
ops = {
  add: fn(a, b) a + b end,
  mul: fn(a, b) a * b end
}
puts ops["add"](2, 3)   # 5 (index access)
puts ops.add(2, 3)      # 5 (dot access — same result)
puts ops.mul(4, 5)      # 20
```

This enables an OOP-like pattern where hashes carry their own methods as closures:

```ruby
def make_record(name)
  record = {name: name}
  record["greet"] = fn() "Hello, " + record.name end
  return record
end

alice = make_record("Alice")
puts alice.greet()   # Hello, Alice
```

## Composing Lambdas

```ruby
compose = fn(f, g)
  return fn(x) f(g(x)) end
end

double = fn(x) x * 2 end
inc = fn(x) x + 1 end
double_then_inc = compose(inc, double)
puts double_then_inc(5)   # 11
```

## Trailing Block Syntax (`do...end`)

When the last argument to a function is a no-argument lambda, you can use `do...end` instead of `fn() ... end`:

```ruby
def with_greeting(block)
  puts "Hello!"
  block()
  puts "Goodbye!"
end

# These are equivalent:
with_greeting(fn()
  puts "Nice to meet you"
end)

with_greeting do
  puts "Nice to meet you"
end
```

If the function takes other arguments, `do...end` appends the block:

```ruby
def repeat(n, block)
  for i in n
    block()
  end
end

repeat(3) do
  puts "hip hip hooray!"
end
```

Nesting works naturally:

```ruby
outer do
  inner("hello") do
    puts "deep"
  end
end
```

Use `fn(params)` when the lambda needs parameters — `do...end` is only for parameterless blocks.

---
Next: [Shell Commands](09-shell.md)
