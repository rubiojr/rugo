# os

Shell execution and process control.

```ruby
use "os"
```

## exec

Runs a shell command and returns its stdout as a string (trailing newline stripped).

```ruby
hostname = os.exec("hostname")
puts hostname

files = os.exec("ls | wc -l")
```

Panics if the command exits non-zero. Use `try` to handle failures:

```ruby
result = try os.exec("might_fail") or "default"
```

> **Tip:** For most cases, prefer backticks instead of `os.exec` — they don't
> require an import:
> ```ruby
> hostname = `hostname`
> result = try `might_fail` or "default"
> ```

## exit

Terminates the program with the given exit code.

```ruby
os.exit(0)   # success
os.exit(1)   # failure
```
