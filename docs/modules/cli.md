# cli

Build CLI applications with commands, flags, and convention-based dispatch.

```ruby
use "cli"
```

## Quick Example

```ruby
use "cli"

cli.name "greet"
cli.version "1.0.0"
cli.about "A friendly greeter"

cli.cmd "hello", "Say hello to someone"
cli.flag "hello", "name", "n", "Name to greet", "World"

cli.run

def hello(args)
  name = cli.get("name")
  puts "Hello, #{name}!"
end
```

```
$ greet hello
Hello, World!

$ greet hello -n Rugo
Hello, Rugo!

$ greet --help
greet 1.0.0 — A friendly greeter

Usage:
  greet <command> [flags] [args...]

Commands:
  hello  Say hello to someone

Global flags:
  -h, --help      Show help
  -V, --version   Show version
```

## App Metadata

### name

Set the application name (shown in help and version output).

```ruby
cli.name "myapp"
```

### version

Set the version string.

```ruby
cli.version "1.0.0"
```

### about

Set the application description.

```ruby
cli.about "A fantastic tool"
```

## Defining Commands

### cmd

Define a command with a name and description.

```ruby
cli.cmd "hello", "Say hello"
cli.cmd "serve", "Start the server"
```

**Subcommands** — use spaces or colons:

```ruby
# Space-separated (user runs: myapp db migrate)
cli.cmd "db migrate", "Run migrations"
cli.cmd "db seed", "Seed the database"

# Colon notation (user runs: myapp db:migrate)
cli.cmd "db:migrate", "Run migrations"
```

### flag

Define a string flag for a command. Arguments: command name, long name, short name, description, default value.

```ruby
cli.flag "serve", "port", "p", "Port to listen on", "8080"
cli.flag "serve", "host", "H", "Host to bind to", "localhost"
```

### bool_flag

Define a boolean flag for a command. Arguments: command name, long name, short name, description.

```ruby
cli.bool_flag "serve", "verbose", "v", "Enable verbose logging"
```

## Dispatch

### run

Parse `os.Args` and dispatch to the matching handler function. This is the main entry point — place it after all command/flag definitions.

```ruby
cli.run
```

Handler functions are matched by convention — command names map to function names:

| Command | Handler function |
|---------|-----------------|
| `hello` | `def hello(args)` |
| `db migrate` | `def db_migrate(args)` |
| `db:migrate` | `def db_migrate(args)` |
| `my-cmd` | `def my_cmd(args)` |

Spaces, colons, and dashes become underscores.

Each handler receives one argument: the array of positional args (before any `--` separator) after the command and flags. Arguments after a `--` are available separately via `cli.passthrough()`.

```ruby
cli.cmd "add", "Add a todo"
cli.run

def add(args)
  puts "Added: #{args[0]}"
end
```

### parse

Parse arguments without dispatching. Use with `command` and `get` for manual routing.

```ruby
cli.parse
cmd = cli.command()
```

## Reading Values

### get

Get the value of a flag. Returns the string value for string flags, `true`/`false` for bool flags.

```ruby
port = cli.get("port")
verbose = cli.get("verbose")
```

### command

Get the matched command name as a string.

```ruby
cli.parse
cmd = cli.command()
```

### args

Get the positional arguments that appear before a `--` separator, as an array.

```ruby
cli.parse
remaining = cli.args()
```

### passthrough

Get the opaque arguments that appear after a `--` separator, as an array. Everything
after the first `--` is captured verbatim (including flag-shaped values like `-v`),
which makes it easy to forward arguments to another command.

```ruby
# myapp run -- ls -la
cli.parse
cmd = cli.passthrough()   # ["ls", "-la"]
```

`args` and `passthrough` are kept separate, so positionals meant for your command
never get mixed up with arguments meant to be forwarded.

## Help

### help

Print help text and exit. Called automatically when no command is given, or when `--help`/`-h` is passed.

```ruby
if len(cli.args()) == 0
  cli.help
end
```

Auto-handled flags:
- `-h`, `--help` — show help (global or per-command)
- `-V`, `--version` — show version

## Full Example

```ruby
use "cli"

cli.name "todo"
cli.version "0.1.0"
cli.about "A minimal todo manager"

cli.cmd "add", "Add a new todo"
cli.flag "add", "priority", "p", "Priority (low/normal/high)", "normal"

cli.cmd "list", "List all todos"
cli.bool_flag "list", "all", "a", "Include completed"

cli.cmd "db migrate", "Run database migrations"
cli.cmd "db seed", "Seed the database"

cli.run

def add(args)
  prio = cli.get("priority")
  puts "Added: #{args[0]} [#{prio}]"
end

def list(args)
  if cli.get("all")
    puts "All todos..."
  else
    puts "Pending todos..."
  end
end

def db_migrate(args)
  puts "Running migrations..."
end

def db_seed(args)
  puts "Seeding..."
end
```
