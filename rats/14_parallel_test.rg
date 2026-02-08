# RATS: Test parallel block concurrency
# Covers parallel expressions, error handling, edge cases, and syntax errors.
use "test"
use "str"

# --- Positive: basic parallel ---

rats "parallel returns ordered results"
  result = test.run("rugo run rats/fixtures/parallel_basic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "10")
  test.assert_eq(lines[1], "20")
  test.assert_eq(lines[2], "30")
end

rats "parallel with shell commands"
  result = test.run("rugo run rats/fixtures/parallel_shell.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "hello")
  test.assert_eq(lines[1], "world")
end

rats "parallel single expression"
  result = test.run("rugo run rats/fixtures/parallel_single.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "1")
end

rats "parallel nested inside parallel"
  result = test.run("rugo run rats/fixtures/parallel_nested.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "2")
  test.assert_eq(lines[1], "99")
end

# --- Positive: error handling ---

rats "parallel error caught with try/or"
  result = test.run("rugo run rats/fixtures/parallel_try_or.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "caught")
end

# --- Positive: empty body ---

rats "parallel with empty body returns empty array"
  result = test.run("rugo run rats/fixtures/parallel_empty.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "done")
end

# --- Positive: codegen gating ---

rats "parallel-only file imports sync but not time"
  result = test.run("rugo emit rats/fixtures/parallel_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"sync\"")
  test.assert_false(str.contains(result["output"], "\"time\""))
  test.assert_false(str.contains(result["output"], "rugoTask"))
end

# --- Positive: build to binary ---

rats "parallel compiles to native binary"
  result = test.run("rugo build -o " + test.tmpdir() + "/par_bin rats/fixtures/parallel_basic.rg")
  test.assert_eq(result["status"], 0)
  result = test.run(test.tmpdir() + "/par_bin")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "10")
  test.assert_eq(lines[1], "20")
  test.assert_eq(lines[2], "30")
end

# --- Negative: syntax errors ---

rats "parallel missing end is a parse error"
  result = test.run("rugo run rats/fixtures/err_parallel_missing_end.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error:")
  test.assert_contains(result["output"], "end")
end

# --- Negative: panic propagation ---

rats "panic in parallel propagates"
  result = test.run("rugo run rats/fixtures/err_parallel_panic.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "integer divide by zero")
end

rats "parallel error output has no Go stacktrace"
  result = test.run("rugo run rats/fixtures/err_parallel_panic.rg")
  test.assert_neq(result["status"], 0)
  test.assert_false(str.contains(result["output"], "goroutine"))
  test.assert_false(str.contains(result["output"], "panic:"))
end
