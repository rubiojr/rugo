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
