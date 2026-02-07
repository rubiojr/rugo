import "conv"
result = try conv.to_i("bad") or err
  puts("caught: " + err)
  42
end
puts(result)
