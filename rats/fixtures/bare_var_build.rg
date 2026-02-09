# Fixture: bare variable compiles to native binary
def greet(name)
  msg = "Hello, " + name
  return msg
end
puts greet("Rugo")
