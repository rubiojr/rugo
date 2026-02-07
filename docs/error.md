# Error Handling in Rugo

Rugo uses `try/or` for error handling — inspired by V lang's `or` blocks,
Zig's inline `catch`, and Ruby's `rescue` modifier. It compiles to Go's
`defer/recover`.

## Quick Reference

```ruby
# Silent recovery — nil on failure
result = try dangerous_call()

# Default value — use fallback on failure
result = try dangerous_call() or "default"

# Handler block — access the error, run recovery logic
result = try dangerous_call() or err
  puts "failed: " + err
  "fallback"
end
```

## Level 1: Silent Recovery

`try EXPR` evaluates the expression and returns `nil` if it panics.

```ruby
# Returns nil if the command fails
result = try `might-not-exist`

# Fire and forget — don't care about the result
try rm /tmp/old-file
```

Use this when you don't care about failures — you just want the script to
continue.

## Level 2: Default Value

`try EXPR or DEFAULT` returns `DEFAULT` when the expression fails.

```ruby
import "conv"

name = try `whoami` or "unknown"
port = try conv.to_i(`cat port.txt`) or 8080
config = try `cat config.json` or "{}"
```

The default expression is only evaluated when the try expression fails.

## Level 3: Handler Block

`try EXPR or err ... end` runs a handler block when the expression fails.
The error message is available as the named variable. The last expression
in the block becomes the result.

```ruby
result = try `cat data.json` or err
  puts "warning: " + err
  puts "using defaults..."
  "{}"
end
```

The handler block can contain multiple statements:

```ruby
data = try `fetch-data` or err
  puts "fetch failed: " + err
  backup = try `cat backup.json` or "{}"
  backup
end
```

## How It Works

All `try/or` forms compile to Go's `defer/recover` wrapped in an
immediately-invoked function:

```ruby
result = try `cmd` or err
  "fallback"
end
```

Compiles to:

```go
result := func() (r interface{}) {
    defer func() {
        if e := recover(); e != nil {
            err := fmt.Sprint(e)
            r = "fallback"
        }
    }()
    return rugo_capture("cmd")
}()
```

The single-line forms (`try EXPR` and `try EXPR or DEFAULT`) are syntactic
sugar — the preprocessor expands them into the block form before parsing.

## Practical Patterns

### Safe file reading

```ruby
config = try `cat config.json` or "{}"
```

### Conversion with fallback

```ruby
import "conv"

num = try conv.to_i("not_a_number") or 0
```

### Conditional on success

```ruby
# try returns nil on failure, which is falsy
result = try `test -f /etc/hosts && echo yes`
if result
  puts "file exists"
end
```

### Nested try

```ruby
data = try `primary-source` or err
  puts "primary failed, trying backup..."
  try `backup-source` or "defaults"
end
```

### Logging errors without stopping

```ruby
try optional-cleanup or err
  puts "cleanup warning: " + err
  nil
end
```

### Catching shell command failures

Since rugo works like bash, shell fallback commands are catchable with `try`:

```ruby
# Shell command that might fail — catch it
try rm /tmp/nonexistent-file
puts "continued after failed rm"

# Get a default when shell fails
user = try whoami or "unknown"

# Handle with error details
result = try cat /etc/shadow or err
  puts "cannot read: " + err
  "access denied"
end
```

Without `try`, failed shell commands exit the script (just like bash).
With `try`, failures are caught and the script continues.

## Limitations

- **Single-line sugar requires `try` at the start of a line or after `=`.**
  Using `try` inside `if` conditions or function arguments requires the full
  block form or assigning to a variable first:

  ```ruby
  # This works — assign first, then check
  result = try `cmd`
  if result
    puts "ok"
  end

  # This also works — full block form in any expression context
  if try `cmd` or _err
    nil
  end
    puts "ok"
  end
  ```
