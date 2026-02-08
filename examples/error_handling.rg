# Error handling in Rugo with try/or
use "conv"

# ============================================================
# Level 1: Silent recovery — try EXPR
# Returns nil on failure, value on success.
# ============================================================

puts "=== Level 1: Silent recovery ==="

# This command doesn't exist — result is nil
result = try `nonexistent_command_xyz_42`
puts "result: " + conv.to_s(result)

# This succeeds — result is the output
result = try `echo works`
puts "result: " + conv.to_s(result)

# Fire and forget — don't even capture the result
try `nonexistent_command_xyz_42`
puts "script continues after failed try"
puts ""

# ============================================================
# Level 2: Default value — try EXPR or DEFAULT
# Returns DEFAULT on failure, value on success.
# ============================================================

puts "=== Level 2: Default value ==="

name = try `whoami` or "unknown"
puts "user: " + name

port = try conv.to_i(`cat /nonexistent/port.txt`) or 8080
puts "port: " + conv.to_s(port)

config = try `cat /etc/nonexistent.conf` or "default_config"
puts "config: " + config
puts ""

# ============================================================
# Level 3: Handler block — try EXPR or err ... end
# Run a handler when the expression fails.
# The error message is available as the named variable.
# The last expression in the block is the result.
# ============================================================

puts "=== Level 3: Handler block ==="

result = try `cat /nonexistent/file` or err
  puts "caught error: " + err
  "fallback_value"
end
puts "result: " + result

# Multi-line handler with logic
data = try `cat /nonexistent/data.json` or err
  puts "warning: " + err
  puts "using defaults..."
  "{}"
end
puts "data: " + data
puts ""

# ============================================================
# Practical examples
# ============================================================

puts "=== Practical examples ==="

# Safe file reading with default
hostname = try `hostname` or "localhost"
puts "hostname: " + hostname

# Chained operations with try
uptime = try `uptime -p` or "uptime unavailable"
puts uptime

# Try in conditions — nil is falsy
result = try `test -f /etc/hosts && echo yes`
if result
  puts "/etc/hosts exists"
end

# Try with conversion
num = try conv.to_i("not_a_number") or 0
puts "parsed number: " + conv.to_s(num)

# Try with shell commands — shell failures are now catchable
try nonexistent_command_42
puts "shell failure caught silently"

result = try nonexistent_command_42 or "shell default"
puts "shell with default: " + result

result = try nonexistent_command_42 or err
  "shell caught: " + err
end
puts result

puts ""
puts "=== All error handling examples passed ==="
