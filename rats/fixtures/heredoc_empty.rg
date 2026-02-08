# Empty heredoc produces empty string
msg = <<TEXT
TEXT
puts ">" + msg + "<"
