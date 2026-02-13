# Benchmarks

Start with the function you want to benchmark.

```ruby
def fib(n)
  if n <= 1
    return n
  end
  return fib(n - 1) + fib(n - 2)
end

puts fib(10)
```

```text
55
```

Run:

```bash
rugo run benchmarks.rugo
```

```text
55
```
