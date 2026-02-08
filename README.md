<p align="center">
  <img src="images/banner.svg" alt="Rugo — Ruby syntax - Shell power - Go binaries" width="700"/>
</p>

# Rugo

Ruby syntax - Shell power - Go binaries.

In a world of software aboundance, agents create your favorite languages.

Will they work? maybe.

Will it burn the planet? perhaps, in the meantime, we'll have great companies.

Can we escape this? In a world currently dominated by software, unlikely.

In a future where code will be written by agents, do we even care about languages? maybe not.

> [!WARNING]
> In case it's not clear enough, Rugo is an agent product.
> Treat it like a ☢️  experiment, totally subject to break.

## Features

- Ruby-like syntax
- Compiles to native binaries — no runtime needed
- [Shell fallback](docs/quickstart/09-shell.md) — unknown commands run as shell commands, like Bash
- [Modules](docs/quickstart/10-modules.md) with namespaces
- [Go stdlib bridge](docs/quickstart/16-go-bridge.md) - call Go standard library functions directly
- [User modules](docs/quickstart/14-custom-modules.md)
- [Error handling](docs/quickstart/11-error-handling.md)
- Built-in [BATS like](https://bats-core.readthedocs.io) test support with [rats](docs/rats.md)
- [Concurrency](docs/quickstart/12-concurrency.md)
- Lightweight, Go-like [OOP](docs/quickstart/17-structs.md)
- Built-in [testing](docs/quickstart/13-testing.md) and [benchmarking](docs/quickstart/15-benchmarks.md)

```ruby
use "http"

# Fetch something from the web
body = http.get("https://httpbin.org/get")
puts body

# Shell commands just work
ls -la | head -5

# Functions
def greet(name)
  puts "Hello, #{name}!"
end

greet "World"

# for..in with string interpolation
scores = [90, 85, 72]
for score in scores
  if score >= 90
    puts "#{score} → A"
  else
    puts "#{score} → B"
  end
end

# Hashes and compound assignment
counts = {}
words = ["hello", "world", "hello", "hello", "world"]
for word in words
  if counts[word]
    counts[word] += 1
  else
    counts[word] = 1
  end
end
for k, v in counts
  puts "#{k}: #{v}"
end

# Error handling
hostname = try `hostname` or "localhost"
puts "Running on #{hostname}"

# Concurrency with spawn
task = spawn http.get("https://httpbin.org/get")
puts task.value

# Parallel execution
results = parallel
  http.get("https://httpbin.org/get")
  http.get("https://httpbin.org/headers")
end
puts results[0]
puts results[1]

# Go stdlib bridge — call Go packages directly
import "math"
import "strconv"

puts math.sqrt(144.0)                        # 12

# Error handling with Go bridge
n = try strconv.atoi("not a number") or 0
puts n  # 0

# Inline tests — ignored by rugo run, executed by rugo rats
use "test"
rats "math.sqrt returns correct value"
  test.assert_eq(math.sqrt(144.0), 12.0)
end
```

## Install

```
go install github.com/rubiojr/rugo@latest
```

## Usage

```bash
rugo script.rg            # compile and run
rugo build script.rg      # compile to native binary
rugo rats script.rg       # run inline tests
rugo emit script.rg       # print generated Go code
```

## Documentation

- [Quickstart Guide](docs/quickstart.md)
- [Modules Reference](docs/modules.md)

