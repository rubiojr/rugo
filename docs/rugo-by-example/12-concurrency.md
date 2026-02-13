# Concurrency

Use `spawn` for one background task:

```ruby
task = spawn
  21 * 2
end

puts task.value
```

```text
42
```

Use `parallel` to run many expressions concurrently:

```ruby
results = parallel
  "users"
  "posts"
end

puts results[0]
puts results[1]
```

```text
users
posts
```
