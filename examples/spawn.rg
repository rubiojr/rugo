# Concurrency with spawn in Rugo
use "conv"

# === Fire and forget ===
puts "=== Fire and forget ==="
spawn
  puts "hello from goroutine"
end
# Small delay to let the goroutine print
`sleep 0.1`
puts ""

# === Task with result ===
puts "=== Task with result ==="
task = spawn
  "computed value: 42"
end
result = task.value
puts result
puts ""

# === One-liner sugar ===
puts "=== One-liner sugar ==="
task = spawn conv.to_s(21 * 2)
puts "answer: " + task.value
puts ""

# === Parallel fan-out ===
puts "=== Parallel fan-out ==="
tasks = []
for i in [1, 2, 3]
  t = spawn
    conv.to_s(i * 10)
  end
  tasks = append(tasks, t)
end
for i, t in tasks
  puts "task #{conv.to_s(i)}: " + t.value
end
puts ""

# === Error handling with try/or ===
puts "=== Error handling ==="
task = spawn
  `nonexistent_command_xyz_42 2>/dev/null`
end
result = try task.value or "caught error gracefully"
puts result
puts ""

# === Check done without blocking ===
puts "=== Polling with .done ==="
task = spawn
  `sleep 0.2 && echo finished`
end
count = 0
while !task.done
  count += 1
  `sleep 0.05`
end
puts "polled " + conv.to_s(count) + " times before done"
puts "result: " + task.value
puts ""

# === Timeout with .wait ===
puts "=== Timeout ==="
task = spawn
  `sleep 5`
end
result = try task.wait(1) or "timed out as expected"
puts result
puts ""

puts "=== All concurrency examples passed ==="
