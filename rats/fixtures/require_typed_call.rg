require "require_typed_lib"

result = require_typed_lib.make_from_env()
puts result["token"]

result2 = require_typed_lib.make_literal()
puts result2["token"]
