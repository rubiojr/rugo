use "test"

import "math/rand/v2"

rats "rand.int_n returns value in range"
  n = rand.int_n(10)
  test.assert_true(n >= 0)
  test.assert_true(n < 10)
end

rats "rand.int_n large range"
  n = rand.int_n(1000000)
  test.assert_true(n >= 0)
  test.assert_true(n < 1000000)
end

rats "rand.float64 returns value in range"
  f = rand.float64()
  test.assert_true(f >= 0.0)
  test.assert_true(f < 1.0)
end

rats "rand.n alias works"
  n = rand.n(100)
  test.assert_true(n >= 0)
  test.assert_true(n < 100)
end

rats "rand produces different values across calls"
  a = rand.int_n(1000000)
  b = rand.int_n(1000000)
  c = rand.int_n(1000000)
  # At least 2 of 3 should differ (vanishingly unlikely all same)
  test.assert_true(a != b || b != c)
end
