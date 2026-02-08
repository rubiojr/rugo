use "conv"
results = parallel
  conv.to_s(1 * 10)
  conv.to_s(2 * 10)
  conv.to_s(3 * 10)
end
for r in results
  puts r
end
