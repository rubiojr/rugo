# Module loading with namespaces
import "conv"
require "helpers"

helpers.greet_user("Rugo Developer")

x = helpers.double(21)
puts("double(21) = " + conv.to_s(x))

m = helpers.max(10, 20)
puts("max(10, 20) = " + conv.to_s(m))
