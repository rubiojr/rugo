# Chapter 7: Structuring Your Code

As scripts grow, Rugo gives you three tools for organization: structs for data
types, modules for code reuse, and the Go bridge for accessing Go's standard
library.

## Structs: Data with Identity

Structs give hashes a name and a constructor. They're lightweight — no class
hierarchies, no inheritance. Just named fields and optional methods.

```ruby
struct Dog
  name
  breed
end

rex = Dog("Rex", "Labrador")
puts rex.name
puts rex.breed

rex.name = "Rexy"
puts "Renamed to: #{rex.name}"
puts type_of(rex)
```

```
Rex
Labrador
Renamed to: Rexy
Dog
```

Under the hood, structs are hashes with a `__type__` field. The constructor
creates the hash and sets each field in order. `type_of()` returns the struct
name.

## Struct Methods with Modules

Struct methods shine when combined with the module system. Define the struct
and its methods in one file, then `require` it from another.

**dog.rugo:**
```ruby
struct Dog
  name
  breed
end

def Dog.speak()
  return self.name + " says woof!"
end
```

**main.rugo:**
```ruby
require "dog"

rex = dog.new("Rex", "Labrador")
puts rex.name
puts dog.speak(rex)
```

```
Rex
Rex says woof!
```

Methods use `self` to access the instance. Callers pass the instance as the first
argument through the namespace: `dog.speak(rex)`. The `new()` function is
automatically created as an alias for the constructor.

## Type Introspection

`type_of()` works on every value. Use it for runtime type checking and
debugging.

```ruby
puts type_of("hello")
puts type_of(42)
puts type_of(3.14)
puts type_of(true)
puts type_of(nil)
puts type_of([1, 2])
puts type_of({a: 1})

double = fn(x) x * 2 end
puts type_of(double)
```

```
String
Integer
Float
Bool
Nil
Array
Hash
Lambda
```

For structs, `type_of()` returns the struct name (`Dog`, `User`, etc.) instead of
`Hash`. This lets you build type-aware functions when needed.

## The Go Bridge

`import` gives you direct access to Go's standard library. Function names are
automatically converted from Go's PascalCase to Rugo's snake_case.

```ruby
import "strings"
import "math"
import "strconv"

puts strings.to_upper("hello rugo")
puts strings.contains("hello world", "world")
puts math.sqrt(144.0)

n = try strconv.atoi("42") or 0
puts n
```

```
HELLO RUGO
true
12
42
```

The bridge covers `strings`, `strconv`, `math`, `path/filepath`, `sort`, `os`,
`time`, and `math/rand/v2`. Go functions that return `(T, error)` auto-panic on
error — pair with `try/or` for safe handling.

## Three Import Mechanisms

| Keyword | Purpose | Example |
|---------|---------|---------|
| `use` | Rugo stdlib modules | `use "http"` |
| `import` | Go stdlib bridge | `import "strings"` |
| `require` | User `.rugo` files | `require "helpers"` |

Use `as` to alias any import when namespaces collide:

```ruby
use "os"
import "os" as go_os
```

**Idiom:** Prefer `use` modules (like `str`) for common operations — they're
designed for Rugo's conventions. Reach for `import` when you need something
the Rugo stdlib doesn't cover.
