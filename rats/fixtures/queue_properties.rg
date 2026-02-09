use "queue"
use "conv"

q = queue.new(2)
q.push("a")
q.push("b")
puts conv.to_s(q.size)
puts conv.to_s(q.closed)
