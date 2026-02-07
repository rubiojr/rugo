# Namespace aliasing with require
import "conv"

require "helpers" as "h"

h.greet_user("Rugo User")
x = h.double(10)
puts("h.double(10) = " + conv.to_s(x))
