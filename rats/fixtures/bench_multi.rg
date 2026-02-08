import "bench"

def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

bench "fib(10)"
  fib(10)
end

bench "fib(15)"
  fib(15)
end
