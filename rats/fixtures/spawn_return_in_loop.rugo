# Fixture: return inside loop inside spawn
task = spawn
  items = [1, 2, 3, 4, 5]
  for x in items
    if x == 3
      return x * 10
    end
  end
  return -1
end
puts task.value
