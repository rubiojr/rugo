# Heredoc with quotes, backslashes, and hash characters
msg = <<TEXT
Line with "double quotes"
And backslash: C:\path
Has # not a comment
TEXT
puts msg
