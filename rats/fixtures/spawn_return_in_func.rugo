# Fixture: return in spawn inside a function
def start_task
  task = spawn
    return "from spawn"
  end
  return task.value
end
puts start_task()
