# Constants in Rugo
#
# Identifiers starting with an uppercase letter are constants.
# They can be assigned once but never reassigned.

use "conv"

# Constants — assigned once, immutable
PI = 3.14159
MAX_RETRIES = 5
AppName = "MyApp"

puts("PI = " + conv.to_s(PI))
puts("MAX_RETRIES = " + conv.to_s(MAX_RETRIES))
puts("AppName = " + AppName)

# Regular variables — can be reassigned freely
count = 0
count += 1
puts("count = " + conv.to_s(count))

# Constants work inside functions too
def circle_area(r)
  Pi = 3.14159
  return Pi * r * r
end

puts("area = " + conv.to_s(circle_area(10)))

# Constants and functions in different scopes are independent
def config_limit()
  Limit = 100
  return Limit
end

puts("limit = " + conv.to_s(config_limit()))

# Hash/array bindings are constant, but contents can change
Config = {"host" => "localhost"}
Config["port"] = 8080
puts("host = " + Config["host"])
puts("port = " + conv.to_s(Config["port"]))
