# Concurrency in Rugo

Design document for goroutine-backed concurrency with a Ruby-flavored syntax.
Both `spawn` and `parallel` are fully implemented.

## Design Philosophy

Rugo's concurrency model should follow the same principles as `try/or`:
- **Simple things are simple** — fire-and-forget is one keyword
- **Complex things are possible** — results, errors, and timeouts compose naturally
- **Ruby-inspired surface, Go-powered core** — clean syntax backed by goroutines
- **No new import required** — concurrency is a language primitive, not a module

## The `spawn` Keyword

`spawn` launches a goroutine and returns a **task** handle. Inspired by
Crystal's `spawn` — familiar in the Ruby world, reads like natural English.

### Level 1: Fire and Forget

```ruby
spawn
  puts "working in background"
end
```

When you don't capture the result, it's pure fire-and-forget. The goroutine
runs independently.

### Level 2: Task with Result

```ruby
task = spawn
  http.get("https://api.example.com/data")
end

# .value blocks until done, returns the result
body = task.value
puts body
```

The last expression in the `spawn` block becomes the task's result — same
convention as `try/or` handler blocks.

### Level 3: One-liner Sugar

For single expressions, parenthesis-free one-liner form:

```ruby
task = spawn http.get("https://api.example.com/data")
```

The preprocessor expands this to the block form, exactly like `try EXPR`
expands to `try EXPR or _err nil end`.

### Composing with `try/or`

Error handling composes naturally — a panicking goroutine stores the error
in the task, and `.value` re-raises it:

```ruby
task = spawn
  http.get("https://unreliable-api.com")
end

# Caught cleanly with try/or
body = try task.value or "request failed"

# Or with full handler
body = try task.value or err
  puts "API error: " + err
  "{}"
end
```

## Parallel Fan-Out

### Manual fan-out with `spawn`

```ruby
import "http"

urls = [
  "https://api.example.com/users",
  "https://api.example.com/posts",
  "https://api.example.com/comments"
]

tasks = []
for url in urls
  t = spawn http.get(url)
  tasks = append(tasks, t)
end

for i, t in tasks
  puts "Response #{i}: " + t.value
end
```

### The `parallel` Block

Syntactic sugar for the common "run N things, wait for all" pattern.
Each statement runs in its own goroutine; the block returns an array of
results in order.

```ruby
import "http"

results = parallel
  http.get("https://api.example.com/users")
  http.get("https://api.example.com/posts")
  http.get("https://api.example.com/comments")
end

puts results[0]   # users response
puts results[1]   # posts response
puts results[2]   # comments response
```

`parallel` errors if **any** task panics — the first error propagates.
Compose with `try/or` to handle gracefully:

```ruby
results = try parallel
  http.get("https://api1.example.com")
  http.get("https://api2.example.com")
end or err
  puts "one request failed: " + err
  [nil, nil]
end
```

## Task API

Tasks are opaque objects with dot-method access (like hash values). They
compile to a `rugoTask` struct in the generated Go code.

| Method | Description |
|--------|-------------|
| `task.value` | Block until done, return result (re-raises errors) |
| `task.done` | Non-blocking check: returns `true` if finished |
| `task.wait(seconds)` | Block with timeout; panics on timeout |

### Timeout example

```ruby
task = spawn
  `long-running-command`
end

result = try task.wait(5) or "timed out after 5s"
```

### Polling example

```ruby
task = spawn
  `expensive-computation`
end

while !task.done
  puts "still working..."
  `sleep 1`
end

puts "result: " + task.value
```

## How It Compiles

### spawn block → goroutine + task struct

```ruby
task = spawn
  http.get("https://example.com")
end
body = task.value
```

Compiles to:

```go
task := func() *rugoTask {
    t := &rugoTask{done: make(chan struct{})}
    go func() {
        defer func() {
            if e := recover(); e != nil {
                t.err = fmt.Sprint(e)
            }
            close(t.done)
        }()
        t.result = rugo_http_get("https://example.com")
    }()
    return t
}()
body := task.Value()
```

### parallel block → N goroutines + sync.WaitGroup

```ruby
results = parallel
  expr1
  expr2
  expr3
end
```

Compiles to:

```go
results := func() interface{} {
    _results := make([]interface{}, 3)
    var _wg sync.WaitGroup
    var _parErr string
    var _parOnce sync.Once
    _wg.Add(3)
    go func() {
        defer _wg.Done()
        defer func() {
            if e := recover(); e != nil {
                _parOnce.Do(func() { _parErr = fmt.Sprint(e) })
            }
        }()
        _results[0] = /* expr1 */
    }()
    go func() {
        defer _wg.Done()
        defer func() {
            if e := recover(); e != nil {
                _parOnce.Do(func() { _parErr = fmt.Sprint(e) })
            }
        }()
        _results[1] = /* expr2 */
    }()
    go func() {
        defer _wg.Done()
        defer func() {
            if e := recover(); e != nil {
                _parOnce.Do(func() { _parErr = fmt.Sprint(e) })
            }
        }()
        _results[2] = /* expr3 */
    }()
    _wg.Wait()
    if _parErr != "" {
        panic(_parErr)
    }
    return interface{}([]interface{}{_results[0], _results[1], _results[2]})
}()
```

### Runtime support (emitted into generated Go)

```go
type rugoTask struct {
    result interface{}
    err    string
    done   chan struct{}
}

func (t *rugoTask) Value() interface{} {
    <-t.done
    if t.err != "" {
        panic(t.err)
    }
    return t.result
}

func (t *rugoTask) Done() interface{} {
    select {
    case <-t.done:
        return true
    default:
        return false
    }
}

func (t *rugoTask) Wait(seconds int) interface{} {
    select {
    case <-t.done:
        if t.err != "" {
            panic(t.err)
        }
        return t.result
    case <-time.After(time.Duration(seconds) * time.Second):
        panic(fmt.Sprintf("task timed out after %d seconds", seconds))
    }
}
```

## Implementation Details

### 1. `spawn` keyword + `SpawnExpr` AST node

Grammar production:

```
SpawnExpr = "spawn" Body "end" .
```

Added to `Primary` alternatives. Returns a `*rugoTask` at the Go level but
exposed as an opaque `interface{}` to Rugo code. The last expression in
the body becomes the task's result.

`spawn` is added to the keyword list in `preprocess.go`.

**One-liner sugar:** The preprocessor expands `spawn EXPR` (where EXPR is
not `end` on the next line) into `spawn\n  EXPR\nend`, same pattern as
`try EXPR` expansion.

### 2. `parallel` keyword + `ParallelExpr` AST node

Grammar production:

```
ParallelExpr = "parallel" Body "end" .
```

Each statement in the body is lifted into a separate goroutine. The block
returns an array of results. Requires `sync` Go import in the generated code.

### 3. Task method calls: `.value`, `.done`, `.wait(n)`

Dot-method calls on the task object. Compiled as runtime helper calls:

```
task.value   →  rugo_task_value(task)
task.done    →  rugo_task_done(task)
task.wait(5) →  rugo_task_wait(task, 5)
```

Task method dispatch is **always active** (not gated on `hasSpawn`), since
users may call `.value` on a task received from another module. The
`usesTaskMethods` AST scan independently gates runtime emission.

Helpers include type-checking with friendly error messages:
`cannot call .value on int — expected a spawn task`

### 4. Runtime additions

The `rugoTask` struct and helper functions are emitted into the runtime
section of generated Go code (in `writeRuntime()`).

**Import gating:** Three independent flags control imports:
- `hasSpawn` → needs `sync` + `time`
- `hasParallel` → needs `sync` only
- `usesTaskMethods` → needs `sync` + `time`

## Limitations

- **One-liner sugar is line-based.** `spawn EXPR` works at the start of a
  line or after `=`, but not nested inside function calls. Use the block form
  or assign to a variable first:

  ```ruby
  # Won't work — spawn inside append()
  # tasks = append(tasks, spawn http.get(url))

  # Works — assign first
  t = spawn http.get(url)
  tasks = append(tasks, t)

  # Also works — block form
  t = spawn
    http.get(url)
  end
  tasks = append(tasks, t)
  ```

## What We Explicitly Defer

| Feature | Reason |
|---------|--------|
| Channels | Spawn+task covers 90% of scripting concurrency needs. Channels add complexity without proportional value for scripts. |
| Mutexes / locks | Not needed — tasks communicate through return values, not shared state. |
| Select / multiplexing | Can be added later if channels are introduced. |
| Worker pools | Out of scope for a scripting language. Use `spawn` in a loop. |
| Context / cancellation | Could be v2. For now, timeouts via `task.wait(n)` suffice. |

## Implementation Order (Complete)

All items have been implemented:

1. ✅ **`spawn` keyword** — grammar, preprocessor, walker, AST node
2. ✅ **`SpawnExpr` codegen** — IIFE with goroutine + rugoTask
3. ✅ **Task runtime** — `rugoTask` struct, `.value`, `.done`, `.wait(n)` with friendly errors
4. ✅ **Task method dispatch** — always-on (not gated on `hasSpawn`)
5. ✅ **One-liner sugar** — preprocessor expansion of `spawn EXPR`
6. ✅ **`parallel` block** — grammar, codegen with WaitGroup + sync.Once

## Complete Example: Parallel HTTP Fetcher (spawn)

```ruby
import "http"
import "conv"

urls = [
  "https://httpbin.org/delay/1",
  "https://httpbin.org/delay/2",
  "https://httpbin.org/delay/3"
]

puts "Fetching #{conv.to_s(len(urls))} URLs in parallel..."

tasks = []
for url in urls
  t = spawn http.get(url)
  tasks = append(tasks, t)
end

for i, t in tasks
  body = try t.value or err
    "error: " + err
  end
  puts "Response #{conv.to_s(i)}: #{body}"
end

puts "Done!"
```

Without concurrency this takes ~6 seconds (sequential). With `spawn`, all
three requests fly in parallel — total time ~3 seconds.

## Complete Example: Parallel HTTP Fetcher (parallel block)

The same pattern, simpler with `parallel`:

```ruby
import "http"

results = try parallel
  http.get("https://httpbin.org/delay/1")
  http.get("https://httpbin.org/delay/2")
  http.get("https://httpbin.org/delay/3")
end or err
  puts "a request failed: " + err
  nil
end

if results != nil
  for i, body in results
    puts body
  end
end
```

## Complete Example: Background Worker

```ruby
# Process items with a background logger
task = spawn
  `tail -f /var/log/app.log`
end

# Do main work while log tails in background
for item in ["a", "b", "c"]
  puts "processing: " + item
end

# Don't wait — just exit (fire-and-forget goroutine)
puts "main work done"
```

## Why `spawn`?

| Alternative | Why not |
|-------------|---------|
| `go` | Conflicts with the Go keyword in generated code; looks too Go-like |
| `async/await` | Two keywords instead of one; implies promises, not goroutines |
| `thread` | Ruby uses this, but goroutines aren't threads |
| `task` | Conflicts with the task *result type* — `task = task ...` is confusing |
| `fork` | Implies process forking, not green threads |
| `spawn` ✓ | Crystal precedent, reads naturally, one keyword, no conflicts |
