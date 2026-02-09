# Fixture: bare variable in def body (not last expression)
def double(n)
  result = n * 2
  result
  return result
end
puts double(21)
