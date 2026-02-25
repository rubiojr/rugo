# Case Expressions

The `case/of/end` construct matches a value against multiple branches:

```ruby
status = "ok"

case status
of "ok"
  puts "all good"
of "error", "fail"
  puts "something went wrong"
else
  puts "unknown"
end
```

## Arrow Form

For concise one-liners, use the `->` arrow syntax:

```ruby
day = "Monday"

case day
of "Saturday", "Sunday" -> puts("weekend")
of "Monday" -> puts("start of week")
else -> puts("weekday")
end
```

## Multiple Values

Each `of` branch can match multiple comma-separated values:

```ruby
code = 404

case code
of 200, 201 -> puts("success")
of 301, 302 -> puts("redirect")
of 400, 404 -> puts("client error")
of 500, 502 -> puts("server error")
else -> puts("other")
end
```

## Expression Form

`case` returns the last value of the matched branch, so it works as an expression inside functions:

```ruby
def describe(color)
  case color
  of "red" -> "warm"
  of "blue" -> "cool"
  of "green" -> "natural"
  else -> "unknown"
  end
end

puts describe("red")    # warm
puts describe("blue")   # cool
```

## Assignment Form

`case` can be assigned directly to a variable:

```ruby
status = "ok"

label = case status
of "ok" -> "success"
of "error" -> "failure"
else -> "unknown"
end

puts label   # success
```

Multi-line branches work too — the last expression in the matched branch becomes the result:

```ruby
code = 404

message = case code
of 200
  "all good"
of 404
  "not found"
else
  "other"
end

puts message   # not found
```

When no branch matches and there is no `else`, the result is `nil`:

```ruby
x = case "mystery"
of "a" -> 1
of "b" -> 2
end

puts x   # (prints nothing — x is nil)
```

## Elsif Branches

You can add `elsif` branches for boolean conditions that don't compare against the subject:

```ruby
score = 75

case score
of 100 -> puts("perfect")
of 0 -> puts("zero")
elsif score >= 90
  puts "A"
elsif score >= 80
  puts "B"
else
  puts "C"
end
```

`of` branches are checked first (by equality), then `elsif` conditions, then `else`.

---
Next: [For Loops](07-for-loops.md)
