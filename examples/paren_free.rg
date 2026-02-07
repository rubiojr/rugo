# Parenthesis-free function calls
import "conv"

# These are equivalent:
puts("With parens")
puts "Without parens"

# Multiple args
puts "Hello", "World"

# With variables
name = "Rugo"
puts "Welcome to", name

# Built-in functions
x = 42
print "The answer is: "
puts conv.to_s(x)

# User-defined functions
def greet(name)
  puts "Hello,", name
end

greet "Developer"
greet("also works")
