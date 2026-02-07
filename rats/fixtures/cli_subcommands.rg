import "cli"

cli.name "sub"
cli.cmd "db:migrate", "Run migrations"
cli.cmd "db:seed", "Seed the database"
cli.run

def db_migrate(args)
  puts "migrating"
end

def db_seed(args)
  puts "seeding"
end
