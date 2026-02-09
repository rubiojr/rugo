# Concurrency

Rugo uses `spawn` to run code in the background. Each `spawn` block runs in
its own goroutine and returns a **task** handle you can use to get the result.

## Fire and Forget

```ruby
spawn
  puts "working in background"
end

puts "main continues immediately"
```

## Getting Results

Capture the task and call `.value` to wait for the result:

```ruby
use "http"

task = spawn
  http.get("https://httpbin.org/get")
end

# .value blocks until the task finishes
body = task.value
puts body
```

## One-Liner Form

For single expressions, skip the block:

```ruby
task = spawn http.get("https://httpbin.org/get")
puts task.value
```

## Parallel Requests

Spawn multiple tasks and collect results:

```ruby
use "http"

urls = ["https://httpbin.org/get", "https://httpbin.org/ip"]

tasks = []
for url in urls
  t = spawn http.get(url)
  tasks = append(tasks, t)
end

for t in tasks
  puts t.value
end
```

## Error Handling

If a `spawn` block panics, the error is captured. Calling `.value` re-raises
it — compose with `try/or` for safe handling:

```ruby
task = spawn
  http.get("https://doesnotexist.invalid")
end

body = try task.value or "request failed"
puts body
```

## Check Without Blocking

Use `.done` to check if a task has finished without blocking:

```ruby
task = spawn `sleep 2 && echo done`

while !task.done
  puts "still waiting..."
  `sleep 1`
end

puts task.value
```

## Parallel Block

Run multiple expressions concurrently and wait for all results:

```ruby
use "http"

results = parallel
  http.get("https://api.example.com/users")
  http.get("https://api.example.com/posts")
end

puts results[0]   # users response
puts results[1]   # posts response
```

Each expression runs in its own goroutine. Results are returned in order as
an array. If any expression panics, `parallel` re-raises the first error —
compose with `try/or`:

```ruby
results = try parallel
  `fast-command`
  `slow-command`
end or err
  puts "one failed: " + err
  "fallback"
end
```

## Timeouts

Use `.wait(seconds)` to block with a time limit — panics on timeout:

```ruby
task = spawn `sleep 10`

result = try task.wait(2) or "timed out"
puts result
```

---
That's it! `spawn` gives you goroutine-powered concurrency with a clean,
Ruby-like syntax.

## Queues

For producer-consumer patterns, use the `queue` module:

```ruby
use "queue"
use "conv"

q = queue.new()

spawn
  for i in [1, 2, 3]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts conv.to_s(item)
end)
```

Queues support bounded capacity (`queue.new(10)`), pop with timeout
(`try q.pop(5) or "timeout"`), and properties (`q.size`, `q.closed`).
See the [queue module docs](../modules/queue.md) for pipelines,
backpressure, and more patterns.

---
See the [concurrency design doc](../concurrency.md) for
the full specification.
