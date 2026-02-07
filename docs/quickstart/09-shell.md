# Shell Commands

Unknown commands run as shell commands automatically.

```ruby
ls -la
whoami
date
```

## Pipes

```ruby
echo "hello world" | tr a-z A-Z
ls -la | head -5
```

## Redirects

```ruby
echo "test" > /tmp/output.txt
echo "more" >> /tmp/output.txt
ls /nonexistent 2>/dev/null
```

## Capture Output

Use backticks to capture command output into a variable:

```ruby
hostname = `hostname`
puts "Running on #{hostname}"
```

Backticks run the command and return stdout as a trimmed string.

## Mix Shell and Rugo

```ruby
name = "World"
puts "Hello, #{name}!"
echo "this runs in the shell"
result = `uname -s`
puts "OS: #{result}"
```

## Shell Exit Codes

Failed shell commands exit the script immediately, just like bash. Use `try` to catch failures:

```ruby
# Without try — script exits on failure
rm /tmp/nonexistent-file

# With try — script continues
try rm /tmp/nonexistent-file
puts "still running"
```

See [Error Handling](11-error-handling.md) for the full `try/or` reference.

## Known Limitations

- **`#` comments:** Rugo strips `#` comments before shell fallback detection, so unquoted `#` in shell commands is treated as a comment. Use quotes: `echo "issue #123"` instead of `echo issue #123`.
- **`test` keyword:** The shell command `test` conflicts with Rugo's `test` keyword (used for test blocks). Use `bash -c "test -f /etc/hosts"` as a workaround.
- **Shell variable syntax:** `FOO=bar` is interpreted as a Rugo assignment, not a shell environment variable. Use `bash -c "FOO=bar command"` instead.

---
Next: [Modules](10-modules.md)
