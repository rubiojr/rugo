# Chapter 3: Functions and Lambdas

Rugo has two kinds of callable things: named functions (`def`) and anonymous
lambdas (`fn`). Both are first-class, but they serve different roles.

## Named Functions: The Workhorses

Define functions with `def`. Parentheses are optional for no-argument functions.
Use early returns to keep logic flat.

```ruby
def classify(score)
  if score >= 90
    return "A"
  end
  if score >= 80
    return "B"
  end
  if score >= 70
    return "C"
  end
  return "F"
end

puts classify(95)
puts classify(82)
puts classify(71)
puts classify(55)
```

```
A
B
C
F
```

Prefer early returns over deeply nested `if/elsif` chains. Each condition is a
clear exit point — the function reads top to bottom.

## Lambdas: Functions as Values

Lambdas are anonymous functions using `fn...end`. They can be stored in
variables, passed as arguments, and returned from functions.

```ruby
double = fn(x) x * 2 end
square = fn(x) x * x end

puts double(5)
puts square(4)
```

```
10
16
```

One-line lambdas are perfect for callbacks. The last expression is the implicit
return value — no `return` needed for simple expressions.

## Higher-Order Functions

Pass lambdas to functions for map, filter, and transform patterns. This is how
idiomatic Rugo replaces what other languages do with built-in iterators.

```ruby
def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end

def my_select(f, arr)
  result = []
  for item in arr
    if f(item)
      result = append(result, item)
    end
  end
  return result
end

nums = [1, 2, 3, 4, 5, 6]
doubled = my_map(fn(x) x * 2 end, nums)
puts doubled

evens = my_select(fn(x) x % 2 == 0 end, nums)
puts evens
```

```
[2 4 6 8 10 12]
[2 4 6]
```

Build these once, use them everywhere. Real Rugo libraries attach `each`, `map`,
and `select` as methods on collection objects (see Chapter 8).

## Closures Capture by Reference

When a lambda references a variable from its enclosing scope, it captures it.
This is the foundation of factory functions.

```ruby
def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
add10 = make_adder(10)
puts add5(3)
puts add10(3)
```

```
8
13
```

Each call to `make_adder` creates a new closure with its own `n`. The lambda
remembers the value even after `make_adder` returns.

## Lambda Dispatch Tables

Store lambdas in hashes for clean dispatch logic. Dot access makes the
calls look like method invocations.

```ruby
ops = {
  add: fn(a, b) a + b end,
  sub: fn(a, b) a - b end,
  mul: fn(a, b) a * b end
}

puts ops.add(10, 3)
puts ops.sub(10, 3)
puts ops.mul(10, 3)
```

```
13
7
30
```

This pattern replaces `switch` statements in other languages. Need a new
operation? Add another entry to the hash.

## Function Composition

Combine small lambdas into larger ones. This is a natural fit for data
transformation pipelines.

```ruby
double = fn(x) x * 2 end
inc = fn(x) x + 1 end

def compose(f, g)
  return fn(x) f(g(x)) end
end

double_then_inc = compose(inc, double)
puts double_then_inc(5)
```

```
11
```

`compose(inc, double)` reads as "increment after doubling" — `double(5)` gives
10, then `inc(10)` gives 11.

## Break and Next in Loops

Use `break` to exit early when you've found what you need. Use `next` to skip
items that don't matter.

```ruby
# Find the first negative number
nums = [3, 7, 2, -1, 5, -3]
for n in nums
  if n < 0
    puts "First negative: #{n}"
    break
  end
end

# Skip nils in a dataset
data = [1, nil, 3, nil, 5]
total = 0
for item in data
  if item == nil
    next
  end
  total += item
end
puts "Total: #{total}"
```

```
First negative: -1
Total: 9
```

These two keywords handle 90% of the cases where other languages need
exceptions, iterators, or complex loop conditions.
