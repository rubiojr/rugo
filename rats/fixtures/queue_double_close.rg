use "queue"

q = queue.new()
q.close()
result = try q.close() or "already-closed"
puts result
