# sqlite

SQLite database access via Go's `database/sql`. Uses `modernc.org/sqlite`
(pure Go, no CGO) — the driver is bundled automatically in compiled binaries.

## Usage

```ruby
use "sqlite"

conn = sqlite.open(":memory:")
sqlite.exec(conn, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")
sqlite.exec(conn, "INSERT INTO users (name, age) VALUES (?, ?)", "Alice", 30)

rows = sqlite.query(conn, "SELECT * FROM users")
for row in rows
  name = row["name"]
  puts name
end

sqlite.close(conn)
```

## Functions

### sqlite.open(path)

Open a SQLite database. Returns a connection handle.

Use `":memory:"` for in-memory databases, or a file path for persistent storage.

```ruby
conn = sqlite.open(":memory:")
conn = sqlite.open("/tmp/myapp.db")
```

### sqlite.exec(conn, sql, params...)

Execute a SQL statement (CREATE, INSERT, UPDATE, DELETE). Returns the number of
rows affected (0 for DDL statements).

```ruby
sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)")
n = sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
puts n  # 1
```

### sqlite.query(conn, sql, params...)

Execute a SELECT query. Returns an array of hashes (one hash per row). Column
names are the hash keys. Returns an empty array if no rows match.

```ruby
rows = sqlite.query(conn, "SELECT * FROM users WHERE age > ?", 18)
for row in rows
  name = row["name"]
  age = row["age"]
  puts "#{name} is #{age}"
end
```

### sqlite.query_row(conn, sql, params...)

Execute a query and return the first row as a hash, or `nil` if no rows match.

```ruby
user = sqlite.query_row(conn, "SELECT * FROM users WHERE id = ?", 1)
if user
  puts user["name"]
end
```

### sqlite.query_val(conn, sql, params...)

Execute a query and return the first column of the first row as a scalar value,
or `nil` if no rows match. Useful for aggregates.

```ruby
count = sqlite.query_val(conn, "SELECT COUNT(*) FROM users")
puts count
```

### sqlite.close(conn)

Close a database connection.

```ruby
sqlite.close(conn)
```

## Type Mapping

| SQLite type | Rugo type |
|-------------|-----------|
| INTEGER     | Integer   |
| REAL        | Float     |
| TEXT        | String    |
| BLOB        | String    |
| NULL        | nil       |

## Parameterized Queries

Always use `?` placeholders for user data. Parameters are bound safely by the
driver, preventing SQL injection.

```ruby
# Safe — parameters are bound by the driver
sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", user_input)

# Dangerous — never do this
# sqlite.exec(conn, "INSERT INTO t (name) VALUES ('" + user_input + "')")
```

## Transactions

Use `BEGIN`, `COMMIT`, and `ROLLBACK` via `sqlite.exec`:

```ruby
sqlite.exec(conn, "BEGIN")
sqlite.exec(conn, "INSERT INTO accounts (name, balance) VALUES (?, ?)", "Alice", 100)
sqlite.exec(conn, "INSERT INTO accounts (name, balance) VALUES (?, ?)", "Bob", 200)
sqlite.exec(conn, "COMMIT")
```

```ruby
sqlite.exec(conn, "BEGIN")
sqlite.exec(conn, "DELETE FROM accounts")
sqlite.exec(conn, "ROLLBACK")
# accounts table is unchanged
```

## Error Handling

All functions panic on error, integrating naturally with `try/or`:

```ruby
result = try sqlite.exec(conn, "bad sql") or -1
count = try sqlite.query_val(conn, "SELECT COUNT(*) FROM t") or 0
```

## Repository Pattern

Wrap sqlite calls in a struct for lightweight OOP:

```ruby
# user_repo.rg
use "sqlite"

struct UserRepo
  conn
end

def UserRepo.add(name, age)
  sqlite.exec(self.conn, "INSERT INTO users (name, age) VALUES (?, ?)", name, age)
end

def UserRepo.find(id)
  return sqlite.query_row(self.conn, "SELECT * FROM users WHERE id = ?", id)
end

def UserRepo.all()
  return sqlite.query(self.conn, "SELECT * FROM users ORDER BY name")
end

def UserRepo.count()
  return sqlite.query_val(self.conn, "SELECT COUNT(*) FROM users")
end
```

```ruby
# main.rg
use "sqlite"
require "user_repo"

conn = sqlite.open(":memory:")
sqlite.exec(conn, "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, age INTEGER)")

repo = user_repo.new(conn)
user_repo.add(repo, "Alice", 30)

for user in user_repo.all(repo)
  name = user["name"]
  puts name
end

sqlite.close(conn)
```
