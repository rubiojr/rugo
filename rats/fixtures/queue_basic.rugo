use "queue"
use "conv"

q = queue.new()

spawn
  q.push(1)
  q.push(2)
  q.push(3)
  q.close()
end

q.each(fn(item)
  puts conv.to_s(item)
end)
