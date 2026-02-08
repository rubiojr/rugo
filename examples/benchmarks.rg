# Benchmarks in Rugo
import "bench"

# Define functions to benchmark
def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

def array_sum()
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  sum = 0
  for x in arr
    sum = sum + x
  end
  return sum
end

# Benchmark blocks auto-calibrate and report timing
bench "fib(20)"
  fib(20)
end

bench "array sum"
  array_sum()
end

bench "string concatenation"
  s = ""
  i = 0
  while i < 100
    s = s + "x"
    i = i + 1
  end
end
