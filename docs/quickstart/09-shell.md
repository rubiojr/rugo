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

## Pipe Operator

The `|` pipe operator connects shell commands with Rugo functions. The left side's output flows as input to the right side.

```ruby
# Shell output to a function
echo "hello world" | puts

# Chain through module functions
import "str"
echo "hello" | str.upper | puts    # HELLO

# Pipe a value to a shell command's stdin
"hello" | tr a-z A-Z | puts        # HELLO

# Assign piped result
name = echo "rugo" | str.upper
puts name                           # RUGO
```

Shell-to-shell pipes still work as before:

```ruby
echo "hello" | tr a-z A-Z          # handled by the shell natively
```

**Note:** The pipe passes **return values**, not stdout. `puts` and `print` return `nil`, so using them in the middle of a chain is a compile error — always put them at the end:

```ruby
ls | puts | head        # ✗ compile error
ls | head | puts        # ✓ puts at the end
```

## Known Limitations

- **`#` comments:** Rugo strips `#` comments before shell fallback detection, so unquoted `#` in shell commands is treated as a comment. Use quotes: `echo "issue #123"` instead of `echo issue #123`.
- **Shell variable syntax:** `FOO=bar` is interpreted as a Rugo assignment, not a shell environment variable. Use `bash -c "FOO=bar command"` instead.

---
Next: [Modules](10-modules.md)
