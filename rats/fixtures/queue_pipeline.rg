use "queue"
use "conv"

stage1 = queue.new()
stage2 = queue.new()

spawn
  for i in [1, 2, 3]
    stage1.push(i)
  end
  stage1.close()
end

spawn
  stage1.each(fn(n)
    stage2.push(n * 10)
  end)
  stage2.close()
end

stage2.each(fn(result)
  puts conv.to_s(result)
end)
