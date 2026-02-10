# Collection Methods

Arrays and hashes have built-in methods for transforming, filtering, and querying data. No imports needed.

## Transforming

```ruby
nums = [1, 2, 3, 4, 5]

# map — transform each element
doubled = nums.map(fn(x) x * 2 end)
puts doubled    # [2, 4, 6, 8, 10]

# flat_map — map then flatten
pairs = [1, 2, 3].flat_map(fn(x) [x, x * 10] end)
puts pairs    # [1, 10, 2, 20, 3, 30]
```

## Filtering

```ruby
nums = [1, 2, 3, 4, 5]

# filter — keep matching elements
big = nums.filter(fn(x) x > 3 end)
puts big    # [4, 5]

# reject — remove matching elements
small = nums.reject(fn(x) x > 3 end)
puts small    # [1, 2, 3]
```

## Reducing

```ruby
nums = [1, 2, 3, 4, 5]

# reduce — accumulate a result
sum = nums.reduce(0, fn(acc, x) acc + x end)
puts sum    # 15

# sum — shorthand for numeric sum
puts nums.sum()    # 15
```

## Searching

```ruby
nums = [1, 2, 3, 4, 5]

# find — first matching element (nil if none)
found = nums.find(fn(x) x > 3 end)
puts found    # 4

# any — true if any element matches
puts nums.any(fn(x) x > 4 end)    # true

# all — true if all elements match
puts nums.all(fn(x) x > 0 end)    # true

# count — number of matching elements
puts nums.count(fn(x) x > 2 end)    # 3
```

## Utilities

```ruby
words = ["hello", "world", "rugo"]

# join — combine into string
puts words.join(", ")    # hello, world, rugo

# first / last
puts words.first()    # hello
puts words.last()     # rugo

# min / max
puts [3, 1, 4, 1, 5].min()    # 1
puts [3, 1, 4, 1, 5].max()    # 5

# uniq — remove duplicates
puts [1, 2, 2, 3, 1].uniq()    # [1, 2, 3]

# flatten — flatten one level
puts [[1, 2], [3, 4]].flatten()    # [1, 2, 3, 4]

# sort_by — sort with custom key
puts ["banana", "fig", "apple"].sort_by(fn(s) len(s) end)
# [fig, apple, banana]
```

## Slicing

```ruby
nums = [1, 2, 3, 4, 5]

# take — first n elements
puts nums.take(3)    # [1, 2, 3]

# drop — all but first n
puts nums.drop(3)    # [4, 5]

# chunk — split into groups
puts nums.chunk(2)    # [[1, 2], [3, 4], [5]]

# zip — pair elements from two arrays
puts [1, 2, 3].zip(["a", "b", "c"])    # [[1, a], [2, b], [3, c]]
```

## Chaining

Methods return arrays, so they chain naturally:

```ruby
result = [1, 2, 3, 4, 5]
  .filter(fn(x) x > 2 end)
  .map(fn(x) x * 10 end)
  .join(" + ")
puts result    # 30 + 40 + 50
```

## Hash Methods

Hash methods pass `(key, value)` to lambdas:

```ruby
person = {name: "Alice", age: 30, city: "NYC"}

# map — returns array of results
puts person.map(fn(k, v) "#{k}=#{v}" end)

# filter / reject — returns a hash
adults = {alice: 30, bob: 17, carol: 25}
  .filter(fn(k, v) v >= 18 end)
puts adults    # {alice: 30, carol: 25}

# find — returns [key, value] or nil
found = person.find(fn(k, v) v == 30 end)
puts found    # [age, 30]

# keys / values
puts person.keys()
puts person.values()

# merge — combine hashes (second wins on conflicts)
merged = person.merge({email: "alice@test.com"})

# reduce — accumulate over pairs
total = {a: 10, b: 20}.reduce(0, fn(acc, k, v) acc + v end)
puts total    # 30

# any / all / count — work like array versions
puts person.any(fn(k, v) v == 30 end)    # true
puts person.count(fn(k, v) type_of(v) == "String" end)    # 2
```

## Each

Use `each` for iteration with side effects:

```ruby
items = []
[1, 2, 3].each(fn(x)
  items = append(items, x * 10)
end)
puts items    # [10, 20, 30]
```

> **Note:** `for..in` is the primary loop form. Use `each` when you need
> a functional style or want to pass iteration as a callback.

---
Next: [Functions](08-functions.md)
