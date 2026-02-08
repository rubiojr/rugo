# Structs

Rugo structs provide lightweight object-oriented programming using hashes with dot access. They work naturally with the `require` namespace system — a struct file acts as a "class."

## Defining a Struct

```ruby
struct Dog
  name
  breed
end
```

This creates a constructor function `Dog(name, breed)` that returns a hash with those fields, plus a `new()` alias for use with namespaces.

## Dot Access on Hashes

Any hash supports dot notation for field access:

```ruby
person = {"name" => "Alice", "age" => 30}
puts person.name          # Alice
person.name = "Bob"       # write access
puts person.name          # Bob
```

Nested dot access works too:

```ruby
data = {"user" => {"name" => "Alice"}}
puts data.user.name       # Alice
```

## Methods

Define methods on a struct type with `def Type.method()`. The first parameter `self` is added automatically:

```ruby
# dog.rg
struct Dog
  name
  breed
end

def Dog.bark()
  return self.name + " says woof!"
end

def Dog.rename(new_name)
  self.name = new_name
end
```

Methods are called through the namespace after requiring the struct file:

```ruby
require "dog"

rex = dog.new("Rex", "Labrador")
puts dog.bark(rex)            # Rex says woof!
dog.rename(rex, "Rexy")
puts dog.bark(rex)            # Rexy says woof!
```

## Structs with Namespaces

The namespace acts as the "class" — `dog.new()` creates instances, `dog.bark(rex)` calls methods.

## Type Tag

Structs automatically include a `__type__` field for identification:

```ruby
rex = Dog("Rex", "Lab")
puts rex.__type__            # Dog
```

## See Also

- `examples/structs/` for a full working example
- `docs/modules.md` for the require/namespace system
