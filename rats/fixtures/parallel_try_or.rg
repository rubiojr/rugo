import "conv"
results = try parallel
  conv.to_s(42)
  `nonexistent_cmd_xyz_42 2>/dev/null`
end or err
  "caught"
end
puts results
