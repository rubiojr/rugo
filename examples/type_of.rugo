# type_of — universal type introspection

puts type_of("hello")       # String
puts type_of(42)            # Integer
puts type_of(3.14)          # Float
puts type_of(true)          # Bool
puts type_of(nil)           # Nil
puts type_of([1, 2, 3])    # Array
puts type_of({a: 1})       # Hash

# Lambdas
double = fn(x) x * 2 end
puts type_of(double)        # Lambda

# Structs return their name
struct Dog
  name
  breed
end

rex = Dog("Rex", "Lab")
puts type_of(rex)           # Dog
