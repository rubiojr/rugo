# Edge cases for positional function vs shell resolution

# 1. Shell fallback before any function is defined
echo "=== Test 1: Shell fallback before def ==="
whoami
date "+%Y"

# 2. Define a function that shadows a shell command
def echo(msg)
  puts "FUNC_ECHO: " + msg
end

# 3. After def, same name should call the function
echo "should be FUNC_ECHO"

# 4. Forward references inside function bodies
def foo()
  bar "from foo"
end

def bar(msg)
  puts "BAR: " + msg
end

# 5. Call foo — it calls bar via forward ref
foo()

# 6. Paren-free call to user function after def
bar "direct call"

# 7. Builtin always wins regardless of position
puts "=== Test 7: Builtins always work ==="

# 8. Function with if/while inside (nested end tracking)
def countdown(n)
  while n > 0
    if n == 1
      puts "last!"
    end
    n = n - 1
  end
  puts "done counting"
end

countdown(3)

# 9. Shell command after a def with nested blocks
# (verifies end-tracking didn't get confused)
echo "after countdown def — should be FUNC_ECHO"

# 10. Multiple defs, interleaved with shell and function calls
def greet(name)
  puts "Hello, " + name
end

greet "World"

# 11. Define another function, call both
def farewell(name)
  puts "Goodbye, " + name
end

greet "Alice"
farewell "Bob"

# 12. Shell command that looks like a function call with dash args
ls -1 /tmp | head -2

# 13. Bare identifier — unknown → shell
uname

# 14. Bare identifier — known function → no-arg call
def status()
  puts "STATUS: all good"
end

status

# 15. Mutual recursion via forward refs
import "conv"

def is_even(n)
  if n == 0
    return "yes"
  end
  return is_odd(n - 1)
end

def is_odd(n)
  if n == 0
    return "no"
  end
  return is_even(n - 1)
end

puts "is 4 even? " + conv.to_s(is_even(4))
puts "is 3 even? " + conv.to_s(is_even(3))

puts "=== All edge cases passed ==="
