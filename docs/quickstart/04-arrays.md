# Arrays

```ruby
fruits = ["apple", "banana", "cherry"]
puts fruits[0]        # apple
puts len(fruits)      # 3
```

## Append

```ruby
append fruits, "date"
puts len(fruits)      # 4
```

The explicit assignment form also works:

```ruby
fruits = append(fruits, "date")
```

## Index Assignment

```ruby
fruits[1] = "blueberry"
puts fruits[1]        # blueberry
```

## Nested Arrays

```ruby
matrix = [[1, 2], [3, 4]]
puts matrix[0]        # [1, 2]
```

## Slicing

Extract a sub-range with `arr[start, length]` (also works on [strings](03-strings.md#slicing)):

```ruby
numbers = [10, 20, 30, 40, 50]
first_two = numbers[0, 2]   # [10, 20]
middle    = numbers[1, 3]   # [20, 30, 40]
```

Out-of-bounds slices are clamped silently — no panics:

```ruby
numbers[3, 100]   # [40, 50] — clamped to end
numbers[99, 5]    # []       — empty array
```

## Negative Indexing

Use negative indices to count from the end of an array:

```ruby
arr = [10, 20, 30, 40, 50]
puts arr[-1]    # 50 (last element)
puts arr[-2]    # 40 (second-to-last)
```

Negative indexing also works with assignment:

```ruby
arr[-1] = 99
puts arr[4]     # 99
```

## Iterating

```ruby
for fruit in fruits
  puts fruit
end
```

See [For Loops](07-for-loops.md) for more iteration patterns.

## Destructuring

Unpack an array into individual variables with comma-separated targets:

```ruby
a, b, c = [10, 20, 30]
puts a   # 10
puts b   # 20
puts c   # 30
```

This works with any expression that returns an array, including Go bridge functions that return multiple values:

```ruby
import "strings"

before, after, found = strings.cut("key=value", "=")
puts before   # key
puts after    # value
puts found    # true
```

---
Next: [Hashes](05-hashes.md)
