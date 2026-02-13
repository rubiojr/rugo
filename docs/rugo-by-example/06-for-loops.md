# For Loops

Iterate arrays:

```ruby
colors = ["red", "green", "blue"]
for color in colors
  puts color
end
```

```text
red
green
blue
```

Iterate with index:

```ruby
colors = ["red", "green", "blue"]
for i, color in colors
  puts "#{i}: #{color}"
end
```

```text
0: red
1: green
2: blue
```

Use `break` and `next` in loops.
