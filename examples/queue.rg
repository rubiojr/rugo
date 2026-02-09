# Queue Module Examples
# Demonstrates producer-consumer, pipelines, bounded queues,
# timeout handling, and streaming results.

use "queue"
use "conv"

# ── Pattern 1: Basic producer-consumer ──────────────────
puts "--- Producer-Consumer ---"

q = queue.new()

spawn
  for word in ["hello", "world", "from", "queue"]
    q.push(word)
  end
  q.close()
end

q.each(fn(item)
  puts item
end)

# ── Pattern 2: Pipeline (multi-stage) ──────────────────
puts ""
puts "--- Pipeline ---"

input = queue.new()
output = queue.new()

# Stage 1: generate numbers
spawn
  for i in [1, 2, 3, 4, 5]
    input.push(i)
  end
  input.close()
end

# Stage 2: multiply by 10
spawn
  input.each(fn(n)
    output.push(n * 10)
  end)
  output.close()
end

# Stage 3: consume
output.each(fn(result)
  puts conv.to_s(result)
end)

# ── Pattern 3: Bounded queue with backpressure ──────────
puts ""
puts "--- Bounded Queue ---"

q = queue.new(2)

spawn
  for i in [1, 2, 3, 4]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts "processed: #{conv.to_s(item)}"
end)

# ── Pattern 4: Pop with timeout ─────────────────────────
puts ""
puts "--- Timeout ---"

q = queue.new()
result = try q.pop(1) or "timed out after 1s"
puts result

# ── Pattern 5: Properties ──────────────────────────────
puts ""
puts "--- Properties ---"

q = queue.new(10)
q.push("a")
q.push("b")
puts "size: #{conv.to_s(q.size)}"
puts "closed: #{conv.to_s(q.closed)}"
q.close()
puts "closed: #{conv.to_s(q.closed)}"
