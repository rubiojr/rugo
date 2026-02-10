# Redirects and Pipes
# Rugo shell commands support standard shell redirections

# Write to file
echo "hello world" > /tmp/rugo_redir_test.txt
cat /tmp/rugo_redir_test.txt

# Append to file
echo "second line" >> /tmp/rugo_redir_test.txt
cat /tmp/rugo_redir_test.txt

# Pipes
echo "hello pipes" | tr a-z A-Z

# Combine pipes
ls /tmp | head -3 | sort

# Discard stderr (use try since the command fails)
try ls /nonexistent_path 2>/dev/null or ""
puts "stderr was silenced"

# Redirect stderr to file instead
echo "error log" 2>&1 > /tmp/rugo_redir_log.txt

# Use try/or to handle failed commands with redirects
result = try ls /no/such/path 2>/dev/null or "not found"
puts "result: #{result}"
puts "still running after failed redirect"

# Capture output with backticks and pipes
line_count = `ls /tmp | wc -l`
puts "Files in /tmp: #{line_count}"

# Cleanup
rm -f /tmp/rugo_redir_test.txt /tmp/rugo_redir_log.txt
