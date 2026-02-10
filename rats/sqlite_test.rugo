# RATS: sqlite module — tests the Rugo glue, not SQLite itself.
# Validates: type normalization, nil semantics, connection handles,
# error handling integration, parameterized query binding.

use "test"
use "sqlite"

def setup()
  conn = sqlite.open(":memory:")
  sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT, score REAL, data BLOB, flag INTEGER)")
  return conn
end

# --- open/close ---

rats "open returns a connection handle"
  conn = sqlite.open(":memory:")
  test.assert_true(conn != nil)
  sqlite.close(conn)
end

rats "open file-based database"
  path = test.tmpdir() + "/test.db"
  conn = sqlite.open(path)
  sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY)")
  sqlite.close(conn)
end

rats "open with file: URI"
  path = "file:" + test.tmpdir() + "/uri_test.db"
  conn = sqlite.open(path)
  sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY)")
  sqlite.exec(conn, "INSERT INTO t VALUES (1)")
  test.assert_eq(sqlite.query_val(conn, "SELECT id FROM t"), 1)
  sqlite.close(conn)
end

rats "open with mode=memory and cache=shared"
  conn = sqlite.open("file:shared_test?mode=memory&cache=shared")
  sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
  sqlite.exec(conn, "INSERT INTO t (v) VALUES (?)", "shared")
  test.assert_eq(sqlite.query_val(conn, "SELECT v FROM t"), "shared")
  sqlite.close(conn)
end

rats "open with mode=ro fails on non-existent file"
  msg = try sqlite.open("file:no_such_file.db?mode=ro") or e
    "" + e
  end
  test.assert_contains(msg, "sqlite.open:")
end

rats "close on already-closed connection is safe"
  conn = sqlite.open(":memory:")
  sqlite.close(conn)
  # database/sql allows double-close without error
  result = try sqlite.close(conn) or "should not error"
  test.assert_nil(result)
end

# --- exec ---

rats "exec DDL returns 0"
  conn = setup()
  n = sqlite.exec(conn, "CREATE TABLE t2 (id INTEGER PRIMARY KEY)")
  test.assert_eq(n, 0)
  sqlite.close(conn)
end

rats "exec INSERT returns rows affected"
  conn = setup()
  n = sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  test.assert_eq(n, 1)
  sqlite.close(conn)
end

rats "exec UPDATE returns rows affected"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Bob")
  n = sqlite.exec(conn, "UPDATE t SET name = ? WHERE name = ?", "Alicia", "Alice")
  test.assert_eq(n, 1)
  sqlite.close(conn)
end

rats "exec DELETE returns rows affected"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Bob")
  n = sqlite.exec(conn, "DELETE FROM t")
  test.assert_eq(n, 2)
  sqlite.close(conn)
end

rats "exec bad SQL panics with sqlite.exec prefix"
  conn = setup()
  msg = try sqlite.exec(conn, "INVALID SQL") or e
    "" + e
  end
  test.assert_contains(msg, "sqlite.exec:")
  sqlite.close(conn)
end

# --- query ---

rats "query returns array of hashes"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name, score) VALUES (?, ?)", "Alice", 9.5)
  sqlite.exec(conn, "INSERT INTO t (name, score) VALUES (?, ?)", "Bob", 8.0)
  rows = sqlite.query(conn, "SELECT name, score FROM t ORDER BY name")
  test.assert_eq(len(rows), 2)
  test.assert_eq(rows[0]["name"], "Alice")
  test.assert_eq(rows[1]["name"], "Bob")
  sqlite.close(conn)
end

rats "query empty result returns empty array"
  conn = setup()
  rows = sqlite.query(conn, "SELECT * FROM t")
  test.assert_eq(len(rows), 0)
  sqlite.close(conn)
end

# --- type normalization ---

rats "INTEGER columns normalize to int"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "test")
  row = sqlite.query_row(conn, "SELECT id FROM t")
  test.assert_eq(type_of(row["id"]), "Integer")
  sqlite.close(conn)
end

rats "REAL columns normalize to float"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name, score) VALUES (?, ?)", "test", 3.14)
  row = sqlite.query_row(conn, "SELECT score FROM t")
  test.assert_eq(type_of(row["score"]), "Float")
  sqlite.close(conn)
end

rats "TEXT columns normalize to string"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "hello")
  row = sqlite.query_row(conn, "SELECT name FROM t")
  test.assert_eq(type_of(row["name"]), "String")
  test.assert_eq(row["name"], "hello")
  sqlite.close(conn)
end

rats "NULL columns normalize to nil"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "test")
  row = sqlite.query_row(conn, "SELECT score FROM t")
  test.assert_nil(row["score"])
  sqlite.close(conn)
end

# --- query_row ---

rats "query_row returns hash on hit"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  row = sqlite.query_row(conn, "SELECT * FROM t WHERE name = ?", "Alice")
  test.assert_eq(row["name"], "Alice")
  sqlite.close(conn)
end

rats "query_row returns nil on miss"
  conn = setup()
  row = sqlite.query_row(conn, "SELECT * FROM t WHERE name = ?", "nobody")
  test.assert_nil(row)
  sqlite.close(conn)
end

# --- query_val ---

rats "query_val returns scalar"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Bob")
  count = sqlite.query_val(conn, "SELECT COUNT(*) FROM t")
  test.assert_eq(count, 2)
  sqlite.close(conn)
end

rats "query_val returns nil on empty result"
  conn = setup()
  val = sqlite.query_val(conn, "SELECT name FROM t WHERE id = ?", 999)
  test.assert_nil(val)
  sqlite.close(conn)
end

rats "query_val returns string value"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  val = sqlite.query_val(conn, "SELECT name FROM t LIMIT 1")
  test.assert_eq(val, "Alice")
  sqlite.close(conn)
end

# --- parameterized queries ---

rats "params bind string, int, float, nil"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name, score, data) VALUES (?, ?, ?)", "test", 42, nil)
  row = sqlite.query_row(conn, "SELECT * FROM t WHERE name = ?", "test")
  test.assert_eq(row["name"], "test")
  test.assert_nil(row["data"])
  sqlite.close(conn)
end

rats "params prevent SQL injection"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  evil = "'; DROP TABLE t; --"
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", evil)
  rows = sqlite.query(conn, "SELECT * FROM t")
  test.assert_eq(len(rows), 2)
  row = sqlite.query_row(conn, "SELECT name FROM t WHERE name = ?", evil)
  test.assert_eq(row["name"], evil)
  sqlite.close(conn)
end

# --- connection handle safety ---

rats "passing non-connection to exec panics with readable message"
  msg = try sqlite.exec("not a conn", "SELECT 1") or e
    "" + e
  end
  test.assert_contains(msg, "sqlite.exec:")
  test.assert_contains(msg, "expected a connection from sqlite.open")
end

rats "passing non-connection to query panics with readable message"
  msg = try sqlite.query(42, "SELECT 1") or e
    "" + e
  end
  test.assert_contains(msg, "sqlite.query:")
  test.assert_contains(msg, "expected a connection from sqlite.open")
end

# --- multiple connections ---

rats "multiple connections are independent"
  c1 = sqlite.open(":memory:")
  c2 = sqlite.open(":memory:")
  sqlite.exec(c1, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
  sqlite.exec(c2, "CREATE TABLE t (id INTEGER PRIMARY KEY, v TEXT)")
  sqlite.exec(c1, "INSERT INTO t (v) VALUES (?)", "from_c1")
  sqlite.exec(c2, "INSERT INTO t (v) VALUES (?)", "from_c2")
  test.assert_eq(sqlite.query_val(c1, "SELECT v FROM t"), "from_c1")
  test.assert_eq(sqlite.query_val(c2, "SELECT v FROM t"), "from_c2")
  sqlite.close(c1)
  sqlite.close(c2)
end

# --- unicode ---

rats "unicode data round-trips correctly"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "日本語テスト 🚀")
  val = sqlite.query_val(conn, "SELECT name FROM t")
  test.assert_eq(val, "日本語テスト 🚀")
  sqlite.close(conn)
end

# --- transactions ---

rats "transactions via exec BEGIN/COMMIT"
  conn = setup()
  sqlite.exec(conn, "BEGIN")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Alice")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "Bob")
  sqlite.exec(conn, "COMMIT")
  test.assert_eq(sqlite.query_val(conn, "SELECT COUNT(*) FROM t"), 2)
  sqlite.close(conn)
end

rats "transactions via exec ROLLBACK"
  conn = setup()
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "before")
  sqlite.exec(conn, "BEGIN")
  sqlite.exec(conn, "INSERT INTO t (name) VALUES (?)", "rolled_back")
  sqlite.exec(conn, "ROLLBACK")
  test.assert_eq(sqlite.query_val(conn, "SELECT COUNT(*) FROM t"), 1)
  test.assert_eq(sqlite.query_val(conn, "SELECT name FROM t"), "before")
  sqlite.close(conn)
end

# --- try/or integration ---

rats "try/or catches exec error with default value"
  conn = setup()
  result = try sqlite.exec(conn, "INSERT INTO nonexistent VALUES (1)") or -1
  test.assert_eq(result, -1)
  sqlite.close(conn)
end

rats "try/or catches exec error with error message"
  conn = setup()
  msg = try sqlite.exec(conn, "INSERT INTO nonexistent VALUES (1)") or e
    "" + e
  end
  test.assert_contains(msg, "sqlite.exec:")
  test.assert_contains(msg, "nonexistent")
  sqlite.close(conn)
end

rats "try/or catches query error with default value"
  conn = setup()
  result = try sqlite.query(conn, "SELECT * FROM nonexistent") or []
  test.assert_eq(len(result), 0)
  sqlite.close(conn)
end

# --- pragmas ---

rats "pragma journal_mode WAL"
  path = test.tmpdir() + "/wal_test.db"
  conn = sqlite.open(path)
  sqlite.exec(conn, "PRAGMA journal_mode=WAL")
  mode = sqlite.query_val(conn, "PRAGMA journal_mode")
  test.assert_eq(mode, "wal")
  sqlite.close(conn)
end

rats "pragma busy_timeout"
  conn = sqlite.open(":memory:")
  sqlite.exec(conn, "PRAGMA busy_timeout=5000")
  val = sqlite.query_val(conn, "PRAGMA busy_timeout")
  test.assert_eq(val, 5000)
  sqlite.close(conn)
end

rats "pragma foreign_keys"
  conn = sqlite.open(":memory:")
  sqlite.exec(conn, "PRAGMA foreign_keys=ON")
  val = sqlite.query_val(conn, "PRAGMA foreign_keys")
  test.assert_eq(val, 1)
  sqlite.close(conn)
end

rats "pragma query returns rows"
  conn = sqlite.open(":memory:")
  sqlite.exec(conn, "CREATE TABLE t (id INTEGER PRIMARY KEY, name TEXT)")
  rows = sqlite.query(conn, "PRAGMA table_info(t)")
  test.assert_eq(len(rows), 2)
  test.assert_eq(rows[0]["name"], "id")
  test.assert_eq(rows[1]["name"], "name")
  sqlite.close(conn)
end
