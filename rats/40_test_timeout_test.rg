# RATS: Test per-test timeout
# Covers the --timeout flag and RUGO_TEST_TIMEOUT env var.
use "test"
use "str"

# --- Positive: test times out with short deadline ---

rats "test times out with short deadline"
  result = test.run("RUGO_TEST_TIMEOUT=2 rugo rats rats/fixtures/test_timeout.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "timed out")
  test.assert_contains(result["output"], "not ok 2")
  test.assert_contains(result["output"], "ok 1")
end

# --- Positive: test passes without timeout pressure ---

rats "test passes with generous timeout"
  result = test.run("RUGO_TEST_TIMEOUT=60 rugo rats rats/fixtures/test_timeout.rg")
  # The second test sleeps 10s which exceeds 60s, so it should timeout
  # Actually, use a separate fixture for fast tests
  result = test.run("RUGO_TEST_TIMEOUT=60 rugo rats rats/02_variables_test.rg")
  test.assert_eq(result["status"], 0)
end

# --- Positive: timeout disabled with 0 ---

rats "timeout disabled with RUGO_TEST_TIMEOUT=0"
  result = test.run("RUGO_TEST_TIMEOUT=0 rugo rats rats/02_variables_test.rg")
  test.assert_eq(result["status"], 0)
end
