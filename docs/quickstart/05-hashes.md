# Hashes

## Colon Syntax

Use `key: value` for string keys — clean and concise:

```ruby
person = {name: "Alice", age: 30, city: "NYC"}
puts person["name"]   # Alice
puts person.name      # Alice
```

## Arrow Syntax

Use `=>` when keys are expressions (variables, integers, booleans):

```ruby
codes = {404 => "Not Found", 500 => "Server Error"}
key = "greeting"
h = {key => "hello"}   # key is the variable value, not the string "key"
```

Both syntaxes can be mixed:

```ruby
h = {name: "Alice", 42 => "answer"}
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

## Calling Lambdas via Dot Access

Lambdas stored in hashes can be called with dot syntax:

```ruby
ops = {
  add: fn(a, b) a + b end,
  mul: fn(a, b) a * b end
}
puts ops.add(2, 3)   # 5
puts ops.mul(4, 5)   # 20
```

See [Lambdas](08b-lambdas.md) for more on this pattern.

---
Next: [Control Flow](06-control-flow.md)
