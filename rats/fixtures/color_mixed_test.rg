# Fixture: mixed test results (pass, fail, skip) for color output testing
use "test"

rats "this test passes"
  test.assert_eq(1, 1)
end

rats "this test is skipped"
  test.skip("intentional skip")
end

rats "this test fails"
  test.assert_eq(1, 2)
end
