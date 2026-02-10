# Type inference benchmarks
# Measures performance gains from typed codegen vs interface{} boxing
use "bench"

# Typed: pure integer recursive function (fully inferred)
def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

bench "typed fib(25)"
  fib(25)
end

# Typed: integer accumulator loop (fully inferred)
bench "typed integer accumulator"
  sum = 0
  i = 0
  while i < 10000
    sum = sum + i
    i = i + 1
  end
end

# Typed: float accumulator loop (fully inferred)
bench "typed float accumulator"
  sum = 0.0
  i = 0
  while i < 10000
    sum = sum + 1.5
    i = i + 1
  end
end

# Typed: boolean-heavy loop (no rugo_to_bool overhead)
bench "typed boolean conditions"
  i = 0
  count = 0
  while i < 10000
    if i > 5000 && i < 8000
      count = count + 1
    end
    i = i + 1
  end
end

# Typed: nested arithmetic (deep expression trees)
bench "typed nested arithmetic"
  i = 0
  while i < 1000
    x = (i * 3 + 7) * (i - 2) / (i + 1)
    i = i + 1
  end
end

# Mixed: for-in loop forces dynamic (collection element type unknown)
bench "mixed for-in with typed accumulator"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  sum = 0
  j = 0
  while j < 100
    for x in arr
      sum = sum + x
    end
    j = j + 1
  end
end
