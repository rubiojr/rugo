# Chapter 6: Concurrent Thinking

Rugo makes concurrency approachable with two primitives: `spawn` for individual
tasks and `parallel` for fan-out work. Both compile to Go goroutines under the
hood.

## Spawn: Fire and Collect

`spawn` runs an expression in a background goroutine and returns a task handle.
Call `.value` to wait for the result.

```ruby
use "conv"

a = spawn 2 + 2
b = spawn 3 * 3
puts "a = #{conv.to_s(a.value)}"
puts "b = #{conv.to_s(b.value)}"
```

```
a = 4
b = 9
```

Both computations run concurrently. `.value` blocks until the result is ready.
This is the simplest concurrency pattern — launch work, collect results.

## Parallel: Fan-Out, Collect All

When you have multiple independent operations, `parallel` runs them all and
returns an ordered array of results.

```ruby
use "conv"

results = parallel
  1 * 10
  2 * 10
  3 * 10
end

for i, r in results
  puts "result #{conv.to_s(i)}: #{conv.to_s(r)}"
end
```

```
result 0: 10
result 1: 20
result 2: 30
```

Results come back in the same order as the expressions — no matter which
goroutine finishes first. This makes `parallel` predictable and easy to reason
about.

## Polling with .done

Sometimes you don't want to block. Use `.done` to check if a task has
finished without waiting.

```ruby
use "conv"

task = spawn 42
# Wait for it to complete
`sleep 0.1`
puts "done: #{conv.to_s(task.done)}"
puts "value: #{conv.to_s(task.value)}"
```

```
done: true
value: 42
```

This is useful for building progress indicators, non-blocking UIs, or
multiplexing multiple tasks.

## Timeouts with .wait

For operations that might hang, `.wait(seconds)` blocks up to a deadline and
panics on timeout. Pair it with `try/or` for clean handling.

```ruby
task = spawn `sleep 5`
result = try task.wait(1) or "timed out"
puts result
```

```
timed out
```

This pattern is essential for network requests, external processes, and anything
where you can't trust the other side to respond promptly.

## The Concurrency Idiom

Idiomatic concurrent Rugo follows a simple recipe:

1. **Independent work →** `parallel` (fan-out, collect all)
2. **Single background task →** `spawn` + `.value`
3. **Unreliable work →** `spawn` + `try task.wait(n) or fallback`
4. **Fire and forget →** `spawn` with no assignment

Always pair `spawn` with `try/or` when the spawned work can fail. Panics inside
a spawned task are captured and re-raised when you call `.value` — wrapping it
in `try` catches them cleanly.
