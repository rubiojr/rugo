# RATS: Test try/or error handling
use "test"

rats "silent recovery returns nil on failure"
  result = test.run("rugo run rats/fixtures/try_silent.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "<nil>")
end

rats "default value on failure"
  result = test.run("rugo run rats/fixtures/try_default.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "99")
end

rats "handler block on failure"
  result = test.run("rugo run rats/fixtures/try_handler.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "caught:")
  test.assert_contains(result["output"], "42")
end

rats "passes through on success"
  result = test.run("rugo run rats/fixtures/try_success.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "shell fallback with try"
  result = test.run("rugo run rats/fixtures/try_shell.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "continued")
end
