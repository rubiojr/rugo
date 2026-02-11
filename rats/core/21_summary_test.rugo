# RATS: Multi-file summary aggregation
use "test"

# Regression: the grand total summary must parse per-file summaries
# even when they contain ANSI color codes.

rats "multi-file summary aggregates test counts"
  result = test.run("rugo rats rats/fixtures/summary/ 2>&1 | grep 'files,'")
  test.assert_contains(result["output"], "2 files, 3 tests,")
  test.assert_contains(result["output"], "3 passed")
  test.assert_contains(result["output"], "0 failed")
end

rats "multi-file summary aggregates with ANSI colors"
  result = test.run("RUGO_FORCE_COLOR=1 NO_COLOR= rugo rats rats/fixtures/summary/ 2>&1 | grep 'files,'")
  test.assert_contains(result["output"], "2 files, 3 tests,")
  test.assert_contains(result["output"], "3 passed")
  test.assert_contains(result["output"], "0 failed")
end
