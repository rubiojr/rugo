import "cli"

cli.name "myapp"
cli.cmd "db migrate", "Run database migrations"
cli.cmd "db seed", "Seed the database"
cli.cmd "server start", "Start the server"
cli.run

def db_migrate(args)
  puts "migrating"
end

def db_seed(args)
  puts "seeding"
end

def server_start(args)
  puts "starting"
end
