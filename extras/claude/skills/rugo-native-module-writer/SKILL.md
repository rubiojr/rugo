---
name: rugo-native-module-writer
description: Expert in writing native Rugo modules (.rugo libraries). Load when creating multi-file Rugo libraries, designing module APIs, or structuring require/with patterns.
---

# Rugo Native Module Writer

Write multi-file `.rugo` libraries that consumers load via `require` and `with`.

## Directory Layouts

### Flat (simple libraries)

```
rugh/
  client.rugo          # → client namespace
  issue.rugo           # → issue namespace
  repo.rugo            # → repo namespace
  main.rugo            # (optional) entry point for bare require
```

Consumer: `require "rugh" with client, issue`

### Nested with lib/ (larger libraries)

```
gummy/
  gummy.rugo           # barrel file: require "lib/db" etc.
  lib/
    db.rugo            # core module
    sql.rugo           # helper module
    fts.rugo           # extension module
```

Consumer: `require "gummy" with db` — the `with` clause checks the root first, then `lib/` as fallback.

### Which to choose

- **Flat**: when modules are independent or loosely coupled (rugh: client, issue, repo)
- **lib/**: when there's a clear public API module backed by internal helpers (gummy: db is public, sql/fts are internal)

## Module Resolution Rules

### `require "dir"` (bare, no with)

Resolves entry point: `<dirname>.rugo` → `main.rugo` → sole `.rugo` file. A file always takes precedence over a directory of the same name.

### `require "dir" with mod1, mod2`

Each name loads `<name>.rugo` from the directory root. If not found, checks `lib/<name>.rugo`. Root takes precedence over lib/.

### Nested requires pass through

When `commands.rugo` requires `mods/alpha` and `mods/beta`, those namespaces (`alpha`, `beta`) are visible to the file that requires `commands`. They keep their own namespaces — they do NOT get re-namespaced under `commands`.

### Barrel files

A barrel file (e.g., `gummy.rugo`) that only contains `require` statements exposes the inner namespaces to the consumer. The barrel file itself has no namespace unless it defines its own functions.

## API Patterns

### Structs with methods (idiomatic)

Structs are the preferred way to build module APIs:

```ruby
# mydb.rugo
use "sqlite"

struct DB
  conn
end

def DB.exec(sql)
  return sqlite.exec(self.conn, sql)
end

def DB.close()
  sqlite.close(self.conn)
end

def open(path)
  return DB(sqlite.open(path))
end
```

Consumer calls through the namespace: `mydb.open(path)`, `mydb.exec(db, sql)`, `mydb.close(db)`.

### Hash-object pattern (closures)

An alternative when you want method calls directly on the object (without the namespace prefix):

```ruby
def open(path)
  conn = sqlite.open(path)

  handle = {}

  handle["query"] = fn(sql)
    return sqlite.query(conn, sql)
  end

  handle["close"] = fn()
    sqlite.close(conn)
  end

  return handle
end
```

The consumer calls `db.open(path)` to get a handle, then `handle.query(sql)`, `handle.close()`.

### Record pattern — augment data with methods

```ruby
def make_record(conn, table, row)
  row["save"] = fn()
    # update using captured conn, table, row["id"]
  end

  row["delete"] = fn()
    sqlite.exec(conn, "DELETE FROM " + table + " WHERE id = ?", row["id"])
  end

  return row
end
```

### Factory chain

Each layer returns closures: `open()` → handle with `model()` → model with `insert()` → record with `save()`/`delete()`.

## Cross-Module Dependencies

Modules can call functions from sibling modules they require:

```ruby
# lib/db.rugo
require "sql"
require "fts"

def _where(conn, table, conditions)
  clause = sql.build_where(conditions)    # call into sql namespace
  # ...
end
```

Dependencies are resolved at call time, not load time. This means a module must `require` its own dependencies — don't rely on the barrel file having loaded them.

## Namespace Gotchas

### Variable shadows namespace

After assignment, a variable with the same name as a namespace takes over for dot access:

```ruby
require "gummy" with db

# Before assignment: db.open() calls the db namespace function
conn = db.open(":memory:")

# After assignment: db.model() is a dot-call on the conn hash
Users = conn.model("users", {name: "text"})
```

This is correct and intentional — local variables shadow namespaces. Use a different variable name if you need continued access to the namespace.

### Private functions

Prefix with `_` by convention. They compile to Go functions and ARE accessible, but signal "don't use this":

```ruby
def _make_model(conn, name, columns)
  # internal
end

def open(path)
  # public API
end
```

### Namespace conflicts

- A `require` namespace cannot conflict with a `use` module (e.g., can't have both `use "os"` and a require namespace `os`)
- A `require` namespace cannot conflict with an `import` Go bridge alias
- `with` and `as` are mutually exclusive on the same `require`

## Conventions

- One public entry point function per module (e.g., `db.open()`, `client.make()`)
- Return hashes with closure "methods" for stateful APIs
- Put `use` and `require` at the top of each file — they must be top-level
- Document usage in a comment at the top of each module file
- Each module file should declare its own dependencies (don't rely on load order)
- Keep barrel files thin — just `require` statements, no logic

## Testing

Write rats tests that exercise the module through the public API:

```ruby
use "test"

rats "module works end to end"
  tmpdir = test.tmpdir()
  script = <<~SCRIPT
    require "mylib" with core
    result = core.do_thing("input")
    puts(result)
  SCRIPT
  test.write_file(tmpdir + "/test.rugo", script)
  result = test.run("rugo run " + tmpdir + "/test.rugo")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "expected")
end
```

For modules using absolute paths, use `import "path/filepath"` and `filepath.abs()` to construct paths.
