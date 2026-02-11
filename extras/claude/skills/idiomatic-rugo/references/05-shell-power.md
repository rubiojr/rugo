# Chapter 5: Shell Power

Rugo isn't just a programming language — it's also a shell. Unknown identifiers
at the top level run as shell commands automatically. This dual nature is one of
Rugo's defining features.

## Backtick Capture

Backticks run a command and capture its stdout into a variable. Trailing
newlines are stripped.

```ruby
user = `whoami`
puts "Running as: #{user}"

count = `ls | wc -l`
puts "Files: #{count}"
```

```
Running as: <your-username>
Files: <file-count>
```

Use backticks whenever you need a command's output as data. It's cleaner than
piping to a variable in Bash.

## The Pipe Operator

The `|` operator connects commands and functions into a pipeline. Shell output
flows into Rugo functions and vice versa.

```ruby
use "str"

echo "hello world" | str.upper | puts

name = echo "rugo" | str.upper
puts "Language: #{name}"

def exclaim(text)
  return text + "!"
end

echo "wow" | exclaim | puts
```

```
HELLO WORLD
Language: RUGO
wow!
```

Pipes pass *return values*, not stdout. This means you can chain shell commands
with Rugo functions seamlessly. Put `puts` at the end of a chain for display.

## Mixing Paradigms

Rugo and shell coexist in the same file. Use each for what it does best.

```ruby
name = "Rugo"
echo "Shell says hello from #{name}"

date_str = `date +%Y`
puts "Year: #{date_str}"
```

```
Shell says hello from Rugo
Year: 2026
```

String interpolation works in shell commands too — the preprocessor expands
`#{}` before handing the line to the shell.

## When Shell, When Rugo?

The compiler uses a simple rule: if it doesn't recognize an identifier as a
Rugo keyword, variable, or function, it falls back to the shell. This means:

- **Known identifiers** → Rugo code
- **Unknown identifiers** → Shell commands
- **Backticks** → Always shell (captures output)
- **`try` wrapping** → Catches shell failures too

**Tip:** If a shell command shares a name with a function you've defined, the
function wins (after its `def`). Use backticks or the full path for the shell
command in that case.

## Shell Safety with try

Shell commands that fail will exit your script by default. Wrap them in `try`
to handle failures gracefully:

```ruby
# Without try — script exits if command fails
# rm /nonexistent/file

# With try — script continues
try rm /nonexistent/file 2>/dev/null
puts "still running"
```

This is especially important in scripts that interact with external tools —
network commands, file operations, and system utilities can all fail.
