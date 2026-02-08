# RATS: Test inline rats blocks in regular .rg files
use "test"

# Test: rugo run ignores rats blocks and runs normal code
rats "rugo run ignores inline rats blocks"
  result = test.run("rugo run rats/fixtures/inline_tests.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "42")
end

# Test: rugo rats executes inline rats blocks
rats "rugo rats runs inline tests"
  result = test.run("rugo rats rats/fixtures/inline_tests.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "ok")
  test.assert_contains(result["output"], "double works")
end

# Test: rugo rats on a file without rats blocks reports no tests
rats "rugo run on file with only rats blocks outputs nothing extra"
  # The fixture file prints 42 when run normally — rats blocks are skipped
  result = test.run("rugo run rats/fixtures/inline_tests.rg")
  # Output should contain only the normal program output, not test results
  test.assert_eq(result["output"], "42")
end

# Test: rugo rats shows compilation errors
rats "rugo rats reports compilation errors"
  result = test.run("rugo rats rats/fixtures/inline_tests_bad.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "not_equal")
end
