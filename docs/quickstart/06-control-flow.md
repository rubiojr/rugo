# Control Flow

## If / Elsif / Else

```ruby
score = 85

if score >= 90
  puts "A"
elsif score >= 80
  puts "B"
else
  puts "C"
end
```

### Postfix `if`

For one-liners, you can put the condition after the statement:

```ruby
puts "big" if score >= 90
x = 42 if ready
```

## Comparison & Logic

```ruby
if x > 0 && x < 100
  puts "in range"
end

if name == "admin" || name == "root"
  puts "superuser"
end

if !done
  puts "still working"
end
```

Operators: `==`, `!=`, `<`, `>`, `<=`, `>=`, `&&`, `||`, `!`

### `||` and `&&` return values

Like Ruby, `||` returns the first truthy value and `&&` returns the last truthy value (or the first falsy one). This makes `||` great for defaults:

```ruby
name = input || "anonymous"
config = load_config() || {}
```

## While

```ruby
i = 0
while i < 5
  puts i
  i += 1
end
```

---
Next: [For Loops](07-for-loops.md)
