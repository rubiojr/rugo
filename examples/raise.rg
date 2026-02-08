# raise — user-defined errors in Rugo
#
# raise signals an error that stops execution unless caught with try/or.

# Uncaught raise terminates the program:
#   raise("something went wrong")

# Validate inputs in library functions
def greet(name)
  if name == nil
    raise "name is required"
  end
  if name == ""
    raise "name cannot be empty"
  end
  return "Hello, " + name
end

# Handler block — access the error message
msg = try greet(nil) or err
  puts "Error: " + err
  "Hello, stranger"
end
puts msg

# Happy path — no error, value passes through
msg = try greet("Alice") or err
  puts "Error: " + err
  "Hello, stranger"
end
puts msg

# Default value — recover with a fallback
msg = try greet("") or "Hello, nobody"
puts msg

# Paren-free syntax works too
result = try raise "boom" or err
  "caught: " + err
end
puts result
