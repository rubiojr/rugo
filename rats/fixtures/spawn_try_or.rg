task = spawn
  `nonexistent_cmd_xyz_42 2>/dev/null`
end
result = try task.value or "caught"
puts result
