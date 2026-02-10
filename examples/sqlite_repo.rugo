# SQLite + Struct Repository Pattern
#
# Shows how to wrap sqlite in a struct for lightweight OOP.

use "sqlite"
require "sqlite_user_repo"

conn = sqlite.open(":memory:")
sqlite.exec(conn, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")

repo = sqlite_user_repo.new(conn)

sqlite_user_repo.add(repo, "Alice", 30)
sqlite_user_repo.add(repo, "Bob", 25)
sqlite_user_repo.add(repo, "Charlie", 35)

count = sqlite_user_repo.count(repo)
puts "Users: #{count}"

for user in sqlite_user_repo.all(repo)
  name = user["name"]
  age = user["age"]
  puts "  #{name} (#{age})"
end

alice = sqlite_user_repo.find(repo, 1)
if alice
  name = alice["name"]
  puts "Found: #{name}"
end

sqlite.close(conn)
