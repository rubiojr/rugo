# Shell with pipes and mixed rugo/shell

name = "Rugo"
puts "Running shell commands from #{name}..."

# Pipes work
ls -la | head -5

# Shell interpolation via string interpolation in the preprocessor
echo "---"

# Mix shell and rugo — backticks capture output
result = `ls | wc -l`
puts "File count: " + result

# Conditionals with shell
if true
  echo "conditional shell works"
end

# shells out
ping -c 1 -W 1 127.0.0.1

def ping(s)
  puts s
end

# uses the function
ping "google.com"
