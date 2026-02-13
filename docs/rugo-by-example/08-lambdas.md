# Lambdas

Lambdas are first-class values.

```ruby
double = fn(x) x * 2 end
puts double(5)
```

```text
10
```

Closures capture surrounding values:

```ruby
def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
puts add5(10)
```

```text
15
```
