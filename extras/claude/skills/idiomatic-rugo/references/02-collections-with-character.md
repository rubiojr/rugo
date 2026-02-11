# Chapter 2: Collections with Character

Rugo's hashes and arrays are more than data containers — they're the building
blocks for objects, configurations, and APIs. Understanding their idioms is
essential to writing good Rugo.

## Hashes as Lightweight Objects

The colon syntax creates clean, readable hashes with string keys. Combined with
dot access, hashes become lightweight objects without ceremony.

```ruby
person = {name: "Alice", age: 30, city: "NYC"}
puts person.name
puts person.city

person.age = 31
puts "#{person.name} is #{person.age}"
```

```
Alice
NYC
Alice is 31
```

Dot access works on *any* hash — you don't need structs for simple data objects.
This is idiomatic Rugo: start with a hash, graduate to a struct only when you
need constructors and methods.

## Colon vs Arrow: Know the Difference

Colon syntax (`key: value`) converts keys to strings. Arrow syntax (`key => value`)
uses the expression as-is — variables, integers, booleans, anything.

```ruby
status_codes = {200 => "OK", 404 => "Not Found", 500 => "Server Error"}
puts status_codes[200]
puts status_codes[404]

key = "greeting"
config = {key => "hello"}
puts config["greeting"]
```

```
OK
Not Found
hello
```

**Rule of thumb:** Use colon syntax for most hashes (it's cleaner). Switch to
arrow syntax when your keys aren't simple string literals.

## Nested Dot Access

Dot access chains through nested hashes — no brackets needed.

```ruby
app = {
  "config" => {
    "db" => {host: "localhost", port: 5432},
    "cache" => {ttl: 300}
  }
}
puts app.config.db.host
puts app.config.cache.ttl
```

```
localhost
300
```

This makes configuration objects feel natural. Build nested hashes for your app's
config and access values with clean dot notation.

## The Factory Pattern

The most powerful Rugo idiom: functions that return hashes with closures attached.
This gives you objects with methods, encapsulation, and state — all without a
class system.

```ruby
def make_counter(start)
  c = {value: start}

  c["increment"] = fn()
    c.value += 1
  end

  c["decrement"] = fn()
    c.value -= 1
  end

  c["get"] = fn()
    return c.value
  end

  return c
end

counter = make_counter(0)
counter.increment()
counter.increment()
counter.increment()
counter.decrement()
puts "Count: #{counter.get()}"
```

```
Count: 2
```

The closures capture the `c` hash by reference, so mutations through `.increment()`
persist. This is how real Rugo libraries like [Gummy](https://github.com/rubiojr/gummy) (an ORM) and [Rugh](https://github.com/rubiojr/rugh) (a GitHub
client) build their APIs — every "object" is a hash with lambda methods.

## Array Idioms

Rugo arrays support negative indexing and slicing — use them.

```ruby
colors = ["red", "green", "blue"]
puts colors[-1]
puts colors[0, 2]

numbers = [10, 20, 30, 40, 50]
first_two = numbers[0, 2]
puts first_two

# Building arrays with append
names = []
names = append(names, "Alice")
names = append(names, "Bob")
puts len(names)
```

```
blue
[red green]
[10 20]
2
```

Note that `append` returns a new array — you must reassign:
`names = append(names, item)`. This is borrowed from Go's semantics.

## Idiomatic Iteration

The `for..in` loop is Rugo's Swiss army knife for iteration. The two-variable
form destructures into index/value for arrays, or key/value for hashes.

```ruby
# Iterate arrays with index
fruits = ["apple", "banana", "cherry"]
for i, fruit in fruits
  puts "#{i}: #{fruit}"
end

# Iterate hashes with key-value
config = {host: "localhost", port: "8080"}
for k, v in config
  puts "#{k} = #{v}"
end
```

```
0: apple
1: banana
2: cherry
host = localhost
port = 8080
```

Prefer `for..in` over `while` with manual indexing. It's clearer and less
error-prone.
