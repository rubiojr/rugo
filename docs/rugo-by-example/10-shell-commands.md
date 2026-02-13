# Shell Commands

Unknown commands automatically run in the shell.

```ruby
echo "hello"
echo "world"
```

```text
hello
world
```

Capture output with backticks:

```ruby
value = `echo rugo`
puts "Running on #{value}"
```

```text
Running on rugo
```

Pipes also work:

```ruby
echo "hello world" | tr a-z A-Z
```

```text
HELLO WORLD
```
