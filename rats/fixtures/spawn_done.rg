task = spawn
  `echo finished`
end
while !task.done
  `sleep 0.01`
end
puts task.value
