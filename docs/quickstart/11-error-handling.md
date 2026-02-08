# Error Handling

Rugo uses `try` / `or` for error handling. Three levels of control.

## Silent Recovery

`try` alone returns `nil` on failure:

```ruby
result = try `nonexistent_command`
# result is nil — script continues
```

Fire and forget:

```ruby
try nonexistent_command
puts "still running"
```

## Default Value

`try ... or default` returns a fallback on failure:

```ruby
hostname = try `hostname` or "localhost"
puts hostname
```

```ruby
use "conv"

port = try conv.to_i("not_a_number") or 8080
puts port   # 8080
```

## Error Handler Block

`try ... or err ... end` runs a block on failure. The error message is available as the named variable. The last expression in the block becomes the result.

```ruby
data = try `cat /missing/file` or err
  puts "Error: #{err}"
  "fallback"
end
puts data   # fallback
```

Shell commands work with `try` too:

```ruby
result = try nonexistent_command or "default"
puts result   # default
```

## Raising Errors

Use `raise` to signal errors from your own code. It works like Go's `panic()` under the hood and can be caught with `try/or`:

```ruby
raise("something went wrong")
```

Paren-free syntax works too:

```ruby
raise "something went wrong"
```

Use `raise` in functions to validate inputs:

```ruby
def greet(name)
  if name == nil
    raise "name is required"
  end
  return "Hello, " + name
end

msg = try greet(nil) or err
  puts "Error: " + err
  "Hello, stranger"
end
puts msg   # Hello, stranger
```

Called without arguments, `raise` uses a default message:

```ruby
result = try raise() or err
  err
end
puts result   # runtime error
```

---
That's it! You now know enough Rugo to build real scripts. See the [examples/](../../examples/) directory for more.
