# Arrays and Hashes

Arrays keep ordered values; hashes map keys to values.

```ruby
fruits = ["apple", "banana", "cherry"]
fruits = append(fruits, "date")
fruits[1] = "blueberry"
puts fruits[0]
puts fruits[-1]
```

```text
apple
date
```

```ruby
user = {name: "Alice", age: 30}
user["age"] = 31
user["email"] = "alice@example.com"
puts user.name
puts user["email"]
```

```text
Alice
alice@example.com
```
