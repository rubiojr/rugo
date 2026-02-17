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
> Rugo is an agent product, driven by Opus 4.6.
> Treat it like a ☢️  experiment, breakage and rough edges expected.
> Having said that, the language spec is mostly stable now, and I'm working
> towards stabiliizing the core and strengthening the test suite.

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
- [Sandboxing](docs/sandbox.md) — opt-in Landlock kernel security, no root needed (Linux only)

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
* **Landlock** (kernel-native sandboxing)

## Install

```
go install github.com/rubiojr/rugo@latest
```

## Usage

```bash
rugo script.rugo            # compile and run
rugo build script.rugo      # compile to native binary
rugo rats script.rugo       # run inline tests
rugo emit script.rugo       # print generated Go code
```

#### Ruby-like syntax

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

greet("World")

def
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

#### Sandboxing

```ruby
# Restrict filesystem and network with Linux Landlock
sandbox ro: ["/etc"], rox: ["/usr", "/lib"], connect: [443]

result = `cat /etc/os-release`
puts result
# Writing to /tmp or connecting to port 80? Denied.
```

Or from the CLI, without modifying the script:

```bash
rugo run --sandbox --ro /etc --rox /usr script.rugo
```

#### Declarative UIs

With the experimental, Qt backed, [Cute](https://github.com/rubiojr/cute) module:

```ruby
require "github.com/rubiojr/cute@v0.3.1"

cute.app("Counter", 400, 300) do
  count = cute.state(0)

  cute.vbox do
    lbl = cute.label("Clicked: 0 times")
    count.on(fn(v) lbl.set_text("Clicked: #{v} times") end)

    cute.button("Click Me") do
      count.set(count.get() + 1)
    end

    cute.hbox do
      cute.button("Reset") do
        count.set(0)
      end
      cute.button("Quit") do
        cute.quit()
      end
    end
  end

  cute.shortcut("Ctrl+Q", fn() cute.quit() end)
end
```

## Documentation

- [Quickstart Guide](docs/quickstart.md)
- [Modules Reference](docs/modules.md)
- [Sandbox (Landlock)](docs/sandbox.md)

---

*Built by someone who's not a compiler expert — just a curious developer dusting off compiler theory notes from 25 years ago, learning as he goes. Rugo is a labor of love, not a production tool.*

