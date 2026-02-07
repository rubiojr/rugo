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

Two-variable form gives `key, value`:

```ruby
config = {"host" => "localhost", "port" => 3000}
for k, v in config
  puts "#{k} = #{v}"
end
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
Next: [Functions](08-functions.md)
