# Hashes

Key-value pairs with `=>`:

```ruby
person = {"name" => "Alice", "age" => 30, "city" => "NYC"}
puts person["name"]   # Alice
```

## Mutation

```ruby
person["age"] = 31
person["email"] = "alice@example.com"
puts person
```

## Empty Hash

```ruby
counts = {}
counts["hello"] = 1
counts["world"] = 2
puts counts
```

## Iterating

```ruby
for key, value in person
  puts "#{key} => #{value}"
end
```

---
Next: [Control Flow](06-control-flow.md)
