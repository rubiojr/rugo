# Chapter 4: Graceful Failure

Error handling in Rugo is built around one construct: `try/or`. It comes in
three levels of control, from casual to comprehensive. Pick the right level for
each situation.

## Level 1: Silent Recovery

Bare `try` catches errors and returns `nil`. Use it when you genuinely don't
care about the failure.

```ruby
use "conv"

result = try `nonexistent_command_xyz_42 2>/dev/null`
puts "result: #{conv.to_s(result)}"
puts "script continues"
```

```
result: <nil>
script continues
```

The script keeps running. No panic, no crash. The result is simply `nil`, which
you can check with `if result == nil` if needed.

**When to use:** Fire-and-forget operations, optional features, cleanup tasks
where failure is acceptable.

## Level 2: Default Value

Add `or DEFAULT` to provide a fallback. This is the most common form — use it
liberally.

```ruby
use "conv"

hostname = try `hostname` or "localhost"
puts "host: #{hostname}"

port = try conv.to_i("not_a_number") or 8080
puts "port: #{port}"
```

```
host: <your-hostname>
port: 8080
```

This is Rugo's answer to null coalescing. One line, clear intent, no ceremony.

**When to use:** Configuration defaults, parsing with fallbacks, any operation
where you have a sensible alternative.

## Level 3: Error Handler Block

When you need to inspect the error or run recovery logic, use the block form.
The error message is bound to the variable after `or`.

```ruby
data = try `cat /nonexistent/config.json` or err
  puts "warning: #{err}"
  "{}"
end
puts "config: #{data}"
```

```
warning: shell command failed (exit 1): cat /nonexistent/config.json
config: {}
```

The last expression in the block becomes the result. This lets you log the error
*and* provide a fallback in the same construct.

**When to use:** When you need logging, metrics, conditional recovery, or when
the error message matters.

## Raising Errors

Use `raise` to signal errors from your own functions. Callers catch them with
`try/or` — it's the same mechanism all the way through.

```ruby
def divide(a, b)
  if b == 0
    raise "division by zero"
  end
  return a / b
end

result = try divide(10, 0) or "cannot divide"
puts result

result = try divide(10, 2) or "cannot divide"
puts result
```

```
cannot divide
5
```

This creates a clean contract: functions `raise` when they can't fulfill their
purpose, callers decide how to handle it. No exception hierarchies, no checked
vs unchecked — just strings and `try/or`.

## The Idiom: Always Have a Plan

Idiomatic Rugo code doesn't let errors propagate silently and doesn't panic
unnecessarily. The pattern is:

1. **Can fail, don't care →** `try expr`
2. **Can fail, have a default →** `try expr or default`
3. **Can fail, need to react →** `try expr or err ... end`

When you're writing a library, `raise` with clear messages. When you're writing
an application, `try/or` at the boundaries. This keeps the error handling close
to where decisions are made.
