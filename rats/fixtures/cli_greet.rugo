use "cli"

cli.name "greet"
cli.version "1.0.0"
cli.about "A friendly greeter"

cli.cmd "hello", "Say hello to someone"
cli.flag "hello", "name", "n", "Name to greet", "World"

cli.cmd "goodbye", "Say goodbye"
cli.bool_flag "goodbye", "loud", "l", "Use uppercase"

cli.run

def hello(args)
  name = cli.get("name")
  puts "Hello, #{name}!"
end

def goodbye(args)
  loud = cli.get("loud")
  if loud
    puts "GOODBYE!"
  else
    puts "Goodbye!"
  end
end
