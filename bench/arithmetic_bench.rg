# Arithmetic & type-switch overhead benchmarks
# Tracks: interface{} boxing, rugo_add/sub/mul/div type-switch cost,
#          rugo_compare string dispatch overhead
use "bench"

def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

bench "fib(10)"
  fib(10)
end

bench "fib(20)"
  fib(20)
end

bench "fib(25)"
  fib(25)
end

bench "integer arithmetic chain"
  x = 0
  i = 0
  while i < 1000
    x = x + i * 2 - 1
    i = i + 1
  end
end

bench "float arithmetic chain"
  x = 0.0
  i = 0
  while i < 1000
    x = x + 1.5 * 2.0 - 0.5
    i = i + 1
  end
end

bench "comparison heavy loop"
  i = 0
  while i < 1000
    if i > 500
      x = i
    end
    i = i + 1
  end
end
