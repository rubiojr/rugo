# Lambdas — first-class functions in Rugo
#
# Rugo supports anonymous functions (lambdas) using fn...end syntax.
# Lambdas can be stored in variables, passed to functions, returned
# from functions, and stored in data structures.

# Basic lambda
double = fn(x) x * 2 end
puts double(5)  # 10

# Multi-param lambda
add = fn(a, b) a + b end
puts add(3, 4)  # 7

# Zero-arg lambda
greet = fn() "hello" end
puts greet()  # hello

# Multi-line lambda
transform = fn(x)
  result = x * 2
  return "value: #{result}"
end
puts transform(21)  # value: 42

# Passing lambdas to functions
def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end

nums = my_map(fn(x) x * 2 end, [1, 2, 3])
puts nums[0]  # 2
puts nums[1]  # 4
puts nums[2]  # 6

# Returning lambdas from functions (closures)
def make_adder(n)
  return fn(x) x + n end
end

add5 = make_adder(5)
puts add5(10)  # 15

# Lambdas in data structures
ops = {
  "add" => fn(a, b) a + b end,
  "mul" => fn(a, b) a * b end
}
puts ops["add"](2, 3)  # 5
puts ops["mul"](4, 5)  # 20

# Composing lambdas
compose = fn(f, g)
  return fn(x) f(g(x)) end
end

inc = fn(x) x + 1 end
double_then_inc = compose(inc, double)
puts double_then_inc(5)  # 11
