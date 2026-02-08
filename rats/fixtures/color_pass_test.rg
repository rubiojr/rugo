# Fixture: all tests pass, for color output testing
use "test"

rats "passes one"
  test.assert_eq(1, 1)
end

rats "passes two"
  test.assert_true(true)
end
