# Fixture with an invalid test module function call.
use "test"

rats "this should fail to compile"
  test.not_equal("a", "b")
end
