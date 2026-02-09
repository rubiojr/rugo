# Fixture: return inside spawn assigns to task result
task = spawn
  x = "hello"
  return x
end
puts task.value
