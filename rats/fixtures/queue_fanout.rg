use "queue"
use "conv"

q = queue.new()

spawn
  for i in [10, 20, 30]
    q.push(i)
  end
  q.close()
end

results = []
q.each(fn(item)
  results = append(results, item)
end)

for r in results
  puts conv.to_s(r)
end
