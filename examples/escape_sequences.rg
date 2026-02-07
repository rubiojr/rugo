# Escape sequences in Rugo strings
# Supports hex (\xHH) and octal (\OOO) escapes

# Hex escape: ANSI green
puts "\x1b[32mThis is green text\x1b[0m"

# Octal escape: ANSI red
puts "\033[31mThis is red text\033[0m"

# Mix with string interpolation
color = "blue"
puts "\x1b[34m#{color} text\x1b[0m"

# Tab and newline still work
puts "col1\tcol2\tcol3"
puts "line1\nline2"
