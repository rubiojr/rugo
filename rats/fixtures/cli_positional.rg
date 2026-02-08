use "cli"

cli.name "pos"
cli.cmd "echo", "Echo positional args"
cli.run

def echo(args)
  for arg in args
    puts arg
  end
end
