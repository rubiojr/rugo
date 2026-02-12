# eval

Compile and run Rugo code at runtime.

```ruby
use "eval"
```

> **Requires:** Go toolchain (`go`) on PATH. If `go` is not found, eval
> panics with a clear error message.

## run

Compiles and runs a Rugo source string. Returns a hash with `status` (exit
code), `output` (combined stdout/stderr), and `lines` (array of output lines).

```ruby
result = eval.run("puts 1 + 1")
puts result["output"]   # => 2
puts result["status"]   # => 0
```

Multiline programs work with heredocs:

```ruby
source = <<RUGO
def greet(name)
  return "hello " + name
end

puts(greet("world"))
RUGO
result = eval.run(source)
puts result["output"]  # => hello world
```

A non-zero exit code is captured, not raised:

```ruby
result = eval.run("exit(42)")
puts result["status"]  # => 42
```

## file

Compiles and runs a Rugo source file. Optional extra arguments are passed
to the compiled program. Returns the same hash format as `run`.

```ruby
result = eval.file("examples/hello.rugo")
puts result["output"]  # => Hello, World!
```
