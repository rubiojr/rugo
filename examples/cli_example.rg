use "cli"
use "conv"

cli.name "todo"
cli.version "0.1.0"
cli.about "A minimal todo manager"

cli.cmd "add", "Add a new todo"
cli.flag "add", "priority", "p", "Priority (low/normal/high)", "normal"

cli.cmd "list", "List all todos"
cli.bool_flag "list", "all", "a", "Include completed"

cli.cmd "db migrate", "Run database migrations"

cli.run

def add(args)
  if len(args) == 0
    puts "error: provide a todo title"
    cli.help
  end
  prio = cli.get("priority")
  puts "Added: #{args[0]} [#{prio}]"
end

def list(args)
  if cli.get("all")
    puts "All todos (including done)..."
  else
    puts "Pending todos..."
  end
end

def db_migrate(args)
  puts "Running migrations..."
end
