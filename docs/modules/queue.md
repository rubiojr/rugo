# queue

Thread-safe queue for producer-consumer concurrency patterns. Built on Go channels.

## Usage

```ruby
use "queue"
```

## Creating Queues

```ruby
q = queue.new()       # default capacity (1024)
q = queue.new(10)     # bounded, capacity 10
q = queue.new(0)      # unbuffered (synchronous)
```

## Methods

| Method | Description |
|--------|-------------|
| `q.push(val)` | Add item to queue. Blocks if bounded and full |
| `q.pop()` | Remove and return item. Blocks until available. Panics if closed and empty |
| `q.pop(n)` | Remove with timeout of `n` seconds. Panics on timeout |
| `q.close()` | Signal no more items. Panics if already closed |
| `q.each(fn)` | Iterate items with lambda. Blocks until queue is closed and drained |

## Properties

| Property | Description |
|----------|-------------|
| `q.size` | Current number of buffered items |
| `q.closed` | `true` if `close()` was called |

## Patterns

### Producer-Consumer

The bread-and-butter pattern — one goroutine produces, another consumes:

```ruby
use "queue"
use "conv"

q = queue.new()

spawn
  for i in [1, 2, 3, 4, 5]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts conv.to_s(item)
end)
```

### Fan-Out with Streaming Results

Multiple producers push results as they arrive — first finished, first processed:

```ruby
use "queue"
use "http"

q = queue.new()
urls = ["https://example.com/a", "https://example.com/b", "https://example.com/c"]

tasks = []
for url in urls
  t = spawn
    body = http.get(url)
    q.push(body)
  end
  tasks = append(tasks, t)
end

# Close queue when all producers finish
spawn
  for t in tasks
    t.value
  end
  q.close()
end

# Process results as they arrive
q.each(fn(body)
  puts "received: " + body
end)
```

### Pipeline (Multi-Stage)

Chain queues together for multi-stage processing:

```ruby
use "queue"
use "conv"

stage1 = queue.new()
stage2 = queue.new()

# Generate
spawn
  for i in [1, 2, 3, 4, 5]
    stage1.push(i)
  end
  stage1.close()
end

# Transform
spawn
  stage1.each(fn(n)
    stage2.push(n * 10)
  end)
  stage2.close()
end

# Consume
stage2.each(fn(result)
  puts conv.to_s(result)
end)
# Output: 10 20 30 40 50
```

### Bounded Queue (Backpressure)

Bounded queues block producers when full — natural rate limiting:

```ruby
use "queue"
use "conv"

q = queue.new(3)

spawn
  for i in [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts "processing: #{conv.to_s(item)}"
end)
```

### Pop with Timeout

Use `try/or` to handle timeouts cleanly:

```ruby
use "queue"

q = queue.new()

spawn
  `sleep 5`
  q.push("late data")
  q.close()
end

item = try q.pop(2) or "timed out"
puts item   # "timed out"
```

### Manual Pop Loop

When you need more control than `each` provides:

```ruby
use "queue"

q = queue.new()

spawn
  for word in ["hello", "world", "stop", "ignored"]
    q.push(word)
  end
  q.close()
end

while !q.closed || q.size > 0
  item = try q.pop(1) or nil
  if item == nil
    break
  end
  if item == "stop"
    puts "sentinel received"
    break
  end
  puts item
end
```

## Error Handling

All queue errors integrate with `try/or`:

```ruby
use "queue"

# Pop timeout
result = try q.pop(5) or "timed out"

# Push to closed queue
result = try q.push("x") or "queue closed"

# Double close
result = try q.close() or "already closed"
```

## How It Works

Queues compile to Go channels under the hood:

- `queue.new(n)` → `make(chan interface{}, n)`
- `q.push(val)` → `ch <- val`
- `q.pop()` → `<-ch`
- `q.close()` → `close(ch)`
- `q.each(fn)` → `for item := range ch { fn(item) }`

Method dispatch uses the generic `DotCall` interface — no special
codegen required. The queue object returned by `queue.new()` implements
`DotGet` (for properties) and `DotCall` (for methods).
