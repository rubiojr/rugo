# Chapter 8: Patterns in the Wild

This final chapter showcases patterns drawn from real Rugo libraries and
projects. These are the idioms that emerge when you put all the pieces together.

## The Data Pipeline

Collections in Rugo are first-class citizens. Arrays and hashes ship with
built-in methods — `map`, `filter`, `reduce`, `find`, `sort_by`, `flat_map`,
and more. No imports, no wrappers. Chain them into pipelines that read like
a description of what you want.

```ruby
scores = [88, 45, 92, 73, 61, 95, 38, 84]

passing = scores
  .filter(fn(s) s >= 60 end)
  .sort_by(fn(s) -s end)
  .map(fn(s) "#{s}" end)
  .join(", ")

puts "Passing: #{passing}"
puts "Top score: #{scores.max()}"
puts "Average: #{scores.sum() / len(scores)}"
```

```
Passing: 95, 92, 88, 84, 73, 61
Top score: 95
Average: 72
```

Each method returns a new array, so the chain flows naturally. `filter` narrows,
`sort_by` orders (negate for descending), `map` transforms, `join` collapses.
This is the Rugo way: describe the transformation, not the loop.

### Querying Collections

Real-world code often needs to ask questions about data — not just transform it.
The search methods (`find`, `any`, `all`, `count`) handle this cleanly.

```ruby
logs = [
  {level: "error", msg: "disk full"},
  {level: "info", msg: "started"},
  {level: "error", msg: "timeout"},
  {level: "info", msg: "connected"},
  {level: "warn", msg: "slow query"}
]

errors = logs.filter(fn(e) e.level == "error" end)
puts "#{len(errors)} errors found:"
errors.each(fn(e)
  puts("  - #{e.msg}")
end)

has_warnings = logs.any(fn(e) e.level == "warn" end)
puts "Warnings present: #{has_warnings}"
```

```
2 errors found:
  - disk full
  - timeout
Warnings present: true
```

### Flattening Nested Data

When your data has structure — teams with members, orders with line items —
`flat_map` pulls the nested values into a single list.

```ruby
teams = [
  {name: "Alpha", members: ["Alice", "Bob"]},
  {name: "Beta", members: ["Carol", "Dave", "Eve"]}
]

all_members = teams.flat_map(fn(t) t.members end)
puts all_members
puts "Total people: #{len(all_members)}"
```

```
[Alice Bob Carol Dave Eve]
Total people: 5
```

### Hash Pipelines

Hash methods work the same way, but lambdas receive `(key, value)` pairs.
This is especially useful for configuration and environment processing.

```ruby
env = {
  host: "localhost",
  port: "5432",
  db: "myapp",
  debug: "true",
  pool_size: "10"
}

conn = env
  .filter(fn(k, v) k == "host" || k == "port" || k == "db" end)
  .map(fn(k, v) "#{k}=#{v}" end)
  .join(" ")

puts conn
```

```
host=localhost port=5432 db=myapp
```

### Reduce for Accumulation

When no single method fits, `reduce` is your escape hatch. It accumulates
a result across the collection — a counter, a hash, a running total.

```ruby
items = ["apple", "banana", "apple", "cherry", "banana", "apple"]

freq = items.reduce({}, fn(acc, item)
  if acc[item] == nil
    acc[item] = 0
  end
  acc[item] += 1
  return acc
end)

most = items.uniq().sort_by(fn(item) -freq[item] end)
most.each(fn(item)
  puts("#{item}: #{freq[item]}")
end)
```

```
apple: 3
banana: 2
cherry: 1
```

The rule of thumb: reach for `filter`, `map`, and `find` first. Pull out
`reduce` when you need to build something that isn't just a filtered or
transformed version of the input.

## The Builder Pattern

Method chaining works naturally when each method returns `self` (or rather,
the hash). This pattern is great for constructing complex objects step by step.

```ruby
use "str"

def query_builder(table)
  q = {
    __table__: table,
    __conditions__: [],
    __order__: nil,
    __limit__: nil
  }

  q["where"] = fn(condition)
    q.__conditions__ = append(q.__conditions__, condition)
    return q
  end

  q["order_by"] = fn(field)
    q.__order__ = field
    return q
  end

  q["limit"] = fn(n)
    q.__limit__ = n
    return q
  end

  q["to_sql"] = fn()
    sql = "SELECT * FROM " + q.__table__
    if len(q.__conditions__) > 0
      sql += " WHERE " + str.join(q.__conditions__, " AND ")
    end
    if q.__order__ != nil
      sql += " ORDER BY " + q.__order__
    end
    if q.__limit__ != nil
      sql += " LIMIT " + q.__limit__
    end
    return sql
  end

  return q
end

sql = query_builder("users").where("age >= 18").where("active = 1").order_by("name").limit("10").to_sql()
puts sql
```

```
SELECT * FROM users WHERE age >= 18 AND active = 1 ORDER BY name LIMIT 10
```

Each method mutates the hash and returns it, enabling the fluid interface.
The [Gummy](https://github.com/rubiojr/gummy) ORM uses this same pattern for its query building.

## Inline Tests

Rugo borrows from Rust: you can embed tests right next to your code. `rugo run`
ignores them; `rugo rats` executes them. Keep your tests close to what they
test.

```ruby
use "test"

def add(a, b)
  return a + b
end

def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end

puts add(2, 3)
puts factorial(5)

rats "add works correctly"
  test.assert_eq(add(1, 2), 3)
  test.assert_eq(add(-1, 1), 0)
  test.assert_eq(add(0, 0), 0)
end

rats "factorial computes correctly"
  test.assert_eq(factorial(0), 1)
  test.assert_eq(factorial(1), 1)
  test.assert_eq(factorial(5), 120)
end
```

Running normally:
```
5
120
```

Running tests with `rugo rats`:
```
ok 1 - add works correctly
ok 2 - factorial computes correctly
2 tests, 2 passed, 0 failed, 0 skipped
```

This is incredibly useful for library code. The tests document the expected
behavior and catch regressions — right there in the same file.

## Lessons from Real Libraries

After studying Gummy (an ORM) and Rugh (a GitHub API client), some patterns
emerge as the Rugo way:

### The Attach Pattern

Libraries build domain objects by attaching methods to a central hash. Each
domain module calls `attach(client)` to inject its functions:

```ruby
# From the Rugh GitHub client:
# user.attach(gh)
# repo.attach(gh)
# issue.attach(gh)
```

This gives you a clean API (`gh.repos()`, `gh.issues()`) while keeping the
implementation modular.

### Smart Records

Database rows and API responses aren't dumb data — they come with actions.
Insert a record, get back an object that can `.save()` and `.delete()` itself:

```ruby
# From Gummy ORM:
# alice = Users.insert({name: "Alice", age: 30})
# alice.name = "Alicia"
# alice.save()              # persists the change
# alice.delete()            # removes from database
```

The trick is attaching closures at creation time that close over the connection
and table name.

### Convention for Internal State

Use double underscores for internal fields that shouldn't be part of the public API:

```ruby
model = {}
model["__conn__"] = conn        # internal: database connection
model["__table__"] = name       # internal: table name
model["insert"] = fn(attrs)     # public: insert a record
  # ...
end
```

This isn't enforced by the language — it's a convention. But it clearly
separates interface from implementation.

---

## The Rugo Philosophy

After eight chapters, the philosophy boils down to a few principles:

1. **Start simple.** Hashes before structs. Functions before classes.
   Graduate to more structure only when you need it.

2. **Be explicit about failure.** Use `try/or` at the right level.
   Don't let errors propagate silently, and don't panic unnecessarily.

3. **Compose small pieces.** Collection methods, lambdas, and closures combine
   into surprisingly powerful patterns without complex abstractions.

4. **Use the shell.** Don't rewrite `curl` or `grep` in Rugo. Shell out
   for what the shell does best, then process results in Rugo.

5. **Test where you code.** Inline `rats` blocks keep tests and
   implementation together. Use them.

Rugo doesn't try to be everything. It's a sharp tool for a specific job:
scripts and tools that need more structure than Bash but less ceremony than Go.
Write it clean, keep it simple, ship a binary.
