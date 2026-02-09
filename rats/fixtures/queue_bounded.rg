use "queue"
use "conv"

q = queue.new(2)

spawn
  for i in [1, 2, 3, 4, 5]
    q.push(i)
  end
  q.close()
end

q.each(fn(item)
  puts conv.to_s(item)
end)
