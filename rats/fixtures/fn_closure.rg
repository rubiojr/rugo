def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
puts add5(10)
