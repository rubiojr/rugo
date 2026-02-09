# Fixture: bare variable as last expression in fn (lambda)
greet = fn(name)
  msg = "Hello, " + name
  msg
end
puts greet("Rugo")
