use "queue"
use "conv"

q = queue.new()
q.push("closed-pop")
q.close()

# Should be able to pop remaining items after close
item = q.pop()
puts item

# Then pop from closed+empty should fail
result = try q.pop() or "empty-after-close"
puts result
