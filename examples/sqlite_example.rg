# SQLite Example
#
# Demonstrates the sqlite module: open, exec, query, query_row,
# query_val, parameterized queries, and error handling.

use "sqlite"

conn = sqlite.open(":memory:")

# Create a table
sqlite.exec(conn, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")

# Insert rows with parameterized queries
sqlite.exec(conn, "INSERT INTO users (name, age) VALUES (?, ?)", "Alice", 30)
sqlite.exec(conn, "INSERT INTO users (name, age) VALUES (?, ?)", "Bob", 25)
sqlite.exec(conn, "INSERT INTO users (name, age) VALUES (?, ?)", "Charlie", 35)

# Query all rows — returns array of hashes
rows = sqlite.query(conn, "SELECT * FROM users ORDER BY name")
for row in rows
  name = row["name"]
  age = row["age"]
  puts "#{name} is #{age}"
end

# Single row — returns hash or nil
user = sqlite.query_row(conn, "SELECT * FROM users WHERE name = ?", "Alice")
if user
  name = user["name"]
  age = user["age"]
  puts "Found: #{name}, age #{age}"
end

# Miss returns nil
nobody = sqlite.query_row(conn, "SELECT * FROM users WHERE name = ?", "Nobody")
if !nobody
  puts "Not found, as expected"
end

# Single value — aggregates
count = sqlite.query_val(conn, "SELECT COUNT(*) FROM users")
puts "Total users: #{count}"

# Error handling with try/or
result = try sqlite.exec(conn, "INSERT INTO nonexistent VALUES (1)") or err
  puts "Caught: #{err}"
  0
end

sqlite.close(conn)

# --- Connection strings ---

# file: URI with create mode
conn2 = sqlite.open("file::memory:?cache=shared")
sqlite.exec(conn2, "CREATE TABLE kv (key TEXT PRIMARY KEY, value TEXT)")
sqlite.exec(conn2, "INSERT INTO kv VALUES (?, ?)", "lang", "rugo")
val = sqlite.query_val(conn2, "SELECT value FROM kv WHERE key = ?", "lang")
puts "Shared memory DB: #{val}"
sqlite.close(conn2)

# --- Pragmas ---

import "os"
path = os.getenv("HOME") + "/.cache/rugo_example.db"
conn3 = sqlite.open(path)

# WAL mode for better concurrency
sqlite.exec(conn3, "PRAGMA journal_mode=WAL")

# 5 second busy timeout
sqlite.exec(conn3, "PRAGMA busy_timeout=5000")

# Enable foreign key enforcement
sqlite.exec(conn3, "PRAGMA foreign_keys=ON")

# Verify pragmas
mode = sqlite.query_val(conn3, "PRAGMA journal_mode")
timeout = sqlite.query_val(conn3, "PRAGMA busy_timeout")
fk = sqlite.query_val(conn3, "PRAGMA foreign_keys")
puts "journal_mode=#{mode}, busy_timeout=#{timeout}, foreign_keys=#{fk}"

sqlite.close(conn3)
os.remove(path)

puts "Done!"
