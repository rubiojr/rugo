import "conv"
result = try conv.to_i("bad") or 99
puts(result)
