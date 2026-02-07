import "conv"
tasks = []
for i in [1, 2, 3]
  t = spawn
    conv.to_s(i * 10)
  end
  tasks = append(tasks, t)
end
for t in tasks
  puts t.value
end
