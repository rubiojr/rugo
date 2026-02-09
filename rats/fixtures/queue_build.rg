use "queue"
use "conv"

q = queue.new()

spawn
  q.push("hello")
  q.push("world")
  q.close()
end

q.each(fn(item)
  puts item
end)
puts "done"
