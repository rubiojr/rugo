task = spawn
  "hello"
end
result = task.wait(5)
puts result
