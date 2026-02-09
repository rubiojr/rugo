# Fixture: bare variable as last expression in spawn
task = spawn
  x = "hello"
  x
end
puts task.value
