# Math utility library.
# Demonstrates Rugo doc comment conventions.

use "conv"

# Calculates the factorial of n.
# Returns 1 when n <= 1.
def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

# Greets a user by name.
def greet(name)
  return "Hello, " + name + "!"
end

# --- Usage ---
# Run: rugo examples/documented.rg
# Run: rugo doc examples/documented.rg
# Run: rugo doc examples/documented.rg factorial

puts greet("world")
puts "5! = " + conv.to_s(factorial(5))
