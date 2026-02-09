use "queue"

q = queue.new()
result = try q.pop(1) or "timed out"
puts result
