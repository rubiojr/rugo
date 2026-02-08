# Squiggly heredoc strips common indentation
name = "World"
msg = <<~HTML
  <h1>Hello #{name}</h1>
  <p>Welcome</p>
HTML
puts msg
