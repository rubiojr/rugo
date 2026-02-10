# Structs Example — lightweight object orientation in Rugo
#
# Structs provide hash-based objects with:
# - Named constructors: Dog("Rex", "Lab") or dog.new("Rex", "Lab")
# - Dot access for fields: rex.name, rex.breed
# - Methods with implicit self: def Dog.bark() ... end
# - Full namespace support via require

require "dog"

# Create instances via namespace
rex = dog.new("Rex", "Labrador")
luna = dog.new("Luna", "Poodle")

# Dot access on struct fields
puts("Name: " + rex.name)
puts("Breed: " + rex.breed)

# Method calls via namespace
puts(dog.bark(rex))
puts(dog.describe(luna))

# Methods can modify self
dog.rename(rex, "Rexy")
puts(dog.bark(rex))

# Dot access works on plain hashes too
config = {"host" => "localhost", "port" => 8080}
puts("Server: " + config.host)

# Nested dot access
data = {"user" => {"name" => "Alice", "email" => "alice@example.com"}}
puts("User: " + data.user.name)
