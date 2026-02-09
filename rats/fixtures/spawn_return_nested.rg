# Fixture: return in nested spawn (inner spawn)
task = spawn
  inner = spawn
    return "inner"
  end
  return "outer:" + inner.value
end
puts task.value
