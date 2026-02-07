# Backtick shell capture example
# Backticks run a shell command and capture its stdout as a string.

# Simple capture
name = `whoami`
puts "User: " + name

# Pipes work inside backticks
file_count = `ls /tmp | wc -l`
puts "Files in /tmp: " + file_count

# Use with expressions
puts "Hostname: " + `hostname`

# Error handling with try
config = try `cat /etc/nonexistent.conf` or "default_config"
puts "Config: " + config
