# For Loops

## Array Iteration

```ruby
colors = ["red", "green", "blue"]
for color in colors
  puts color
end
```

## With Index

Two-variable form gives `index, value`:

```ruby
for i, color in colors
  puts "#{i}: #{color}"
end
```

## Hash Iteration

Single-variable form gives **keys**:

```ruby
config = {"host" => "localhost", "port" => 3000}
for k in config
  puts k
end
# prints host, port
```

Two-variable form gives `key, value`:

```ruby
for k, v in config
  puts "#{k} = #{v}"
end
```

## Integer Ranges

Iterate from `0` to `N-1`:

```ruby
for i in 5
  puts i
end
# prints 0, 1, 2, 3, 4
```

Use `range(start, end)` to iterate from `start` to `end-1`:

```ruby
for i in range(3, 7)
  puts i
end
# prints 3, 4, 5, 6
```

`range()` also works as a standalone function that returns an array:

```ruby
arr = range(5)
# arr is [0, 1, 2, 3, 4]
```

## Break

Stop the loop early:

```ruby
for n in [1, 2, 3, 4, 5]
  if n == 4
    break
  end
  puts n
end
# prints 1, 2, 3
```

## Next

Skip to the next iteration:

```ruby
for n in [1, 2, 3, 4, 5]
  if n % 2 == 0
    next
  end
  puts n
end
# prints 1, 3, 5
```

`break` and `next` work in `while` loops too.

---
Next: [Collection Methods](07b-collection-methods.md)
