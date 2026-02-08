# Heredoc inside a function
def greet(name)
  msg = <<~TEXT
    Hello #{name}
    Welcome!
  TEXT
  return msg
end

puts greet("Rugo")
