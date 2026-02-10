# Inline Tests
#
# Rugo supports embedding tests directly in regular .rg files.
# When run with `rugo run`, the rats blocks are ignored.
# When run with `rugo rats`, only the rats blocks are executed.

use "test"

def add(a, b)
  return a + b
end

def greet(name)
  return "Hello, " + name + "!"
end

puts add(2, 3)
puts greet("World")

# Inline tests — ignored by `rugo run`, executed by `rugo rats`
rats "add returns the sum"
  test.assert_eq(add(1, 2), 3)
  test.assert_eq(add(-1, 1), 0)
  test.assert_eq(add(0, 0), 0)
end

rats "greet formats a greeting"
  test.assert_eq(greet("Rugo"), "Hello, Rugo!")
  test.assert_contains(greet("World"), "World")
end
