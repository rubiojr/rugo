<p align="center">
  <img src="images/banner.svg" alt="Rugo — Ruby syntax - Shell power - Go binaries" width="700"/>
</p>

# Rugo

Ruby syntax - Shell power - Go binaries.

In a world of software aboundance, agents create your favorite languages.

Will they work? maybe.

Will it burn the planet? perhaps, [in the meantime, we'll have great companies](https://www.youtube.com/watch?v=YE5adUeTe_I).

Can we escape this? In a world currently dominated by software, unlikely.

In a future where code will be written by agents, do we even care about languages? maybe not.

> [!WARNING]
> In case it's not clear enough, Rugo is an agent product, driven by Opus 4.6.
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
- [Lambdas](docs/quickstart/08b-lambdas.md) — first-class anonymous functions
- Lightweight, Go-like [OOP](docs/quickstart/17-structs.md)
- Built-in [testing](docs/quickstart/13-testing.md) and [benchmarking](docs/quickstart/15-benchmarks.md)

## Influences

Rugo stands on the shoulders of giants:

* **Ruby** (syntax, blocks)
* **Go** (compilation, structs)
* **Crystal** (spawn concurrency)
* **V** (try/or error handling)
* **Zig** (inline catch)
* **Bash** (shell fallback, pipes)
* **BATS** (test runner)
* **Rust** (inline tests alongside code).
* **Elixir** (Lambdas)

#### Ruby-like syntax

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

greet("World")

scores = [90, 85, 72]
for score in scores
  if score >= 90
    puts "#{score} → A"
  else
    puts "#{score} → B"
  end
end
```

#### Shell fallback

```ruby
ls -la | head -3
name = `whoami`
puts "I'm #{name}"
```

#### Lambdas

```ruby
double = fn(x) x * 2 end
puts double(5)

add = fn(a, b) a + b end
puts add(2, 3)
```

#### Modules

```ruby
use "str"
use "conv"

puts str.upper("hello rugo")
puts conv.to_i("42") + 8
```

#### Go stdlib bridge

```ruby
import "math"
import "strings"

puts math.sqrt(144.0)
puts strings.to_upper("hello")
```

#### Error handling

```ruby
import "strconv"

hostname = try `hostname` or "localhost"
puts "Running on #{hostname}"

n = try strconv.atoi("nope") or 0
puts n
```

#### Concurrency

```ruby
a = spawn 2 + 2
b = spawn 3 * 3

puts a.value
puts b.value
```

#### Structs

```ruby
struct Dog
  name
  breed
end

rex = Dog("Rex", "Labrador")
puts rex.name
puts rex.breed
```

#### Inline tests

```ruby
use "test"

def add(a, b)
  return a + b
end

rats "add works"
  test.assert_eq(add(2, 3), 5)
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

---

*Built by someone who's not a compiler expert — just a curious developer dusting off compiler theory notes from 25 years ago, learning as he goes. Rugo is a labor of love, not a production tool.*

