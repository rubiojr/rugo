# signal

Race-free OS signal handling. Catch signals like `SIGINT` (Ctrl-C) and
`SIGTERM` to shut down cleanly, reload configuration on `SIGHUP`, and more.

## Usage

```ruby
use "signal"
```

## Why "race-free"?

Handlers registered with `signal.on` run on the goroutine that calls
`signal.wait()` — **not** on the OS signal-delivery goroutine. While a handler
runs, `wait()` is busy and your program's main flow is parked inside it, so a
handler never runs concurrently with the code that installed it. This is the
key difference from raw OS signal callbacks, which fire on a separate thread
and can corrupt shared state.

## Functions

| Function | Description |
|----------|-------------|
| `signal.on(name, handler)` | Register a handler for a named signal. The handler runs on the `wait()` goroutine |
| `signal.wait()` | Block the calling goroutine, dispatching handlers as signals arrive. Does not return |
| `signal.reset(name)` | Stop invoking the handler for a signal (undoes `signal.on`) |
| `signal.ignore(name)` | Ignore a signal so it no longer affects the process |

## Signal Names

Names are case-insensitive and the `SIG` prefix is optional (`"INT"` and
`"SIGINT"` are equivalent).

| Name | Signal | Typical use |
|------|--------|-------------|
| `INT` | SIGINT | Ctrl-C — interactive interrupt |
| `TERM` | SIGTERM | Polite termination (default `kill`) |
| `HUP` | SIGHUP | Reload configuration |
| `QUIT` | SIGQUIT | Quit (often with a core dump) |
| `USR1` | SIGUSR1 | User-defined |
| `USR2` | SIGUSR2 | User-defined |
| `WINCH` | SIGWINCH | Terminal window resized |
| `PIPE` | SIGPIPE | Write to a closed pipe |
| `ALRM` | SIGALRM | Timer expiry |

An unknown name raises an error you can catch with `try/or`.

## Graceful Shutdown

The most common pattern — catch Ctrl-C, clean up, exit:

```ruby
use "signal"

signal.on("INT", fn()
  puts "shutting down..."
  exit(0)
end)

puts "running — press Ctrl-C to stop"
signal.wait()
```

`signal.wait()` blocks forever and dispatches handlers as signals arrive. A
handler calls `exit()` to terminate the program.

## Handling Several Signals

Register a handler per signal. `wait()` dispatches whichever arrives:

```ruby
use "signal"

signal.on("INT", fn()
  puts "interrupted"
  exit(130)
end)

signal.on("TERM", fn()
  puts "terminated"
  exit(0)
end)

signal.wait()
```

## Reload on SIGHUP

A handler that does not call `exit()` simply returns, and `wait()` keeps
looping — ready for the next signal. This suits "reload" signals:

```ruby
use "signal"

signal.on("HUP", fn()
  puts "reloading configuration"
end)

signal.on("INT", fn()
  puts "bye"
  exit(0)
end)

signal.wait()
```

## Ignoring and Resetting

```ruby
use "signal"

# Ignore Ctrl-C entirely — the process is unaffected by SIGINT.
signal.ignore("INT")

# Later, stop invoking a handler you previously registered.
signal.on("USR1", fn() puts "got USR1" end)
signal.reset("USR1")
```

`signal.reset` undoes a prior `signal.on` (the signal is no longer forwarded to
Rugo). It does not restore the original OS default disposition for a signal
Rugo has already trapped — use `signal.ignore` when you need the signal to have
no effect.

## Doing Work While Handling Signals

Because handlers run on the `wait()` goroutine, put `signal.wait()` in the main
flow and run your actual work in a `spawn`. Share state through a `queue` (or
another thread-safe primitive), never plain variables — that keeps everything
race-free:

```ruby
use "signal"
use "queue"

jobs = queue.new()

# Worker runs in the background.
spawn
  jobs.each(fn(job)
    puts "processing #{job}"
  end)
end

# On Ctrl-C, close the queue so the worker drains and stops, then exit.
signal.on("INT", fn()
  jobs.close()
  exit(0)
end)

jobs.push("a")
jobs.push("b")

signal.wait()
```

## Error Handling

An unknown signal name integrates with `try/or`:

```ruby
use "signal"

result = try signal.on("NOPE", fn() end) or err
  puts "bad signal: #{err}"
  "handled"
end
```

## How It Works

The module maps friendly names to `os.Signal` values and uses Go's
`os/signal.Notify` under the hood:

- `signal.on(name, fn)` → registers `fn` and calls `signal.Notify(ch, sig)`
- `signal.wait()` → `for sig := range ch { handler[sig]() }` on the caller's goroutine
- `signal.reset(name)` → `signal.Reset(sig)`
- `signal.ignore(name)` → `signal.Ignore(sig)`

The handler map is guarded by a mutex, so registering and dispatching are safe
even across goroutines.
