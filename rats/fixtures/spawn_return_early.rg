# Fixture: early return inside spawn with conditional
task = spawn
  x = 42
  if x > 10
    return "big"
  end
  return "small"
end
puts task.value
