classify = fn(x)
  if x > 0
    return "positive"
  end
  return "non-positive"
end
puts classify(1)
puts classify(-1)
