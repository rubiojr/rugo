# Module loading with namespaces
import "conv"
require "helpers"

helpers.greet_user("Rugo Developer")

x = helpers.double(21)
puts("double(21) = " + conv.to_s(x))

# User modules can import stdlib modules and use them internally
puts(helpers.double_str(21))

m = helpers.max(10, 20)
puts("max(10, 20) = " + conv.to_s(m))
