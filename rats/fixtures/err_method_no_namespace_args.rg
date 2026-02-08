# Calling rename() directly without the namespace should fail.
require "struct_dog" as "dog"

rex = dog.new("Rex", "Labrador")
rename(rex, "Rexy")
