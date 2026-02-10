# Benchmarking

Rugo includes a built-in benchmark framework using the `bench` keyword.
Benchmark blocks auto-calibrate iteration count and report timing results.

## Writing Benchmarks

Create a `.rugo` file with `bench` blocks:

```ruby
use "bench"

def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

bench "fib(20)"
  fib(20)
end

bench "array sum"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  sum = 0
  for x in arr
    sum = sum + x
  end
end
```

## Running Benchmarks

```bash
rugo run benchmarks.rugo          # run a single benchmark file
rugo bench                      # run all _bench.rugo files in current directory
rugo bench bench/               # run all _bench.rugo files in a directory
rugo bench bench/fib_bench.rugo   # run a specific file
```

Output looks like:

```
  fib(20)                                    132.5 µs/op (7626 runs)
  array sum                                  126.0 ns/op (7985354 runs)
```

## How It Works

Each `bench` block is run repeatedly. The framework:

1. **Warms up** with one initial call
2. **Auto-calibrates** — starts with 1 iteration, scales up until total time ≥ 1 second
3. **Reports** nanoseconds per operation and total iterations

## Using `_bench.rugo` Files with `rugo bench`

The `rugo bench` command discovers `_bench.rugo` files in the target
directory. This convention mirrors Go's `_test.go` naming — benchmark
files use the `_bench.rugo` suffix to distinguish them from regular
scripts (`.rugo`) and tests (`_test.rugo`).

```bash
# Create bench/arithmetic_bench.rugo
rugo bench bench/
```

## Top-Level Code

Code outside `bench` blocks runs once before benchmarks start. Use it
for setup:

```ruby
use "bench"

# Runs once: setup shared data
data = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

bench "iterate data"
  sum = 0
  for x in data
    sum = sum + x
  end
end
```

## Functions

Define helper functions alongside benchmarks:

```ruby
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
```

---

That's it! Benchmarks give you a simple way to measure and compare
performance of your Rugo code.
