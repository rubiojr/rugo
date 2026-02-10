# Fixture for inline tests regression test.
# This file has both normal code and rats blocks.

use "test"

def double(x)
  return x * 2
end

puts double(21)

rats "double works"
  test.assert_eq(double(5), 10)
end
