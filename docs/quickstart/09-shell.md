# Shell Commands

Unknown commands run as shell commands automatically.

```ruby
ls -la
whoami
date
```

## Pipes

```ruby
echo "hello world" | tr a-z A-Z
ls -la | head -5
```

## Redirects

```ruby
echo "test" > /tmp/output.txt
echo "more" >> /tmp/output.txt
ls /nonexistent 2>/dev/null
```

## Capture Output

Use backticks to capture command output into a variable:

```ruby
hostname = `hostname`
puts "Running on #{hostname}"
```

Backticks run the command and return stdout as a trimmed string.

## Mix Shell and Rugo

```ruby
name = "World"
puts "Hello, #{name}!"
echo "this runs in the shell"
result = `uname -s`
puts "OS: #{result}"
```

---
Next: [Modules](10-modules.md)
