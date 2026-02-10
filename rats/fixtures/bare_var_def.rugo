# Fixture: bare variable as last expression in def
# Before fix, `msg` was treated as shell command.
# After fix, it evaluates as variable (returns nil without explicit return).
def greet(name)
  msg = "Hello, " + name
  return msg
end
puts greet("Rugo")
