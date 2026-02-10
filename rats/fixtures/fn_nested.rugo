compose = fn(f, g)
  return fn(x) f(g(x)) end
end

double = fn(x) x * 2 end
inc = fn(x) x + 1 end
double_then_inc = compose(inc, double)
puts double_then_inc(5)
