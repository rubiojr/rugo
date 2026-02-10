# Parallel block in Rugo
use "conv"

# Run expressions in parallel, collect ordered results
puts "=== Basic parallel ==="
results = parallel
  conv.to_s(1 * 10)
  conv.to_s(2 * 10)
  conv.to_s(3 * 10)
end

for i, r in results
  puts "result #{conv.to_s(i)}: " + r
end

# Parallel with try/or for error handling
puts ""
puts "=== Parallel with try/or ==="
results = try parallel
  conv.to_s(42)
  `nonexistent_cmd_xyz_42 2>/dev/null`
end or err
  puts "caught: " + err
  "fallback"
end
puts results

# Parallel with shell commands
puts ""
puts "=== Parallel shell ==="
results = parallel
  `echo hello`
  `echo world`
end
puts results[0]
puts results[1]

puts ""
puts "=== All parallel examples passed ==="
