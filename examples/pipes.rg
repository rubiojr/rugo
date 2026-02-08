# Pipe operator examples
# The | operator passes the output of the left side to the right side.
# Shell commands → captured stdout; functions → return value.
# Right side: functions receive piped value as first arg;
#             shell commands receive it on stdin.

use "str"

# Shell output piped to a function
echo "hello pipes" | puts

# Chaining: shell → module function → builtin
echo "hello world" | str.upper | puts

# Expression piped to function
len("hello") | puts

# Value piped to shell stdin then to function
"hello world" | tr a-z A-Z | puts

# Assignment with pipe
name = echo "rugo" | str.upper
puts "Language: #{name}"

# Pipe with user-defined function
def exclaim(text)
  return text + "!"
end

echo "hello" | exclaim | puts

# Shell-to-shell pipes still work as before
echo "mixed case" | tr a-z A-Z
