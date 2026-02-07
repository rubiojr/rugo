# Functions in Rugo
import "conv"

def greet(name)
  puts("Hello, " + name + "!")
end

def add(a, b)
  return a + b
end

def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

def fibonacci(n)
  if n <= 0
    return 0
  end
  if n == 1
    return 1
  end
  return fibonacci(n - 1) + fibonacci(n - 2)
end

greet("World")
greet("Rugo")

result = add(3, 4)
puts("3 + 4 = " + conv.to_s(result))

puts("5! = " + conv.to_s(factorial(5)))
puts("fib(10) = " + conv.to_s(fibonacci(10)))
