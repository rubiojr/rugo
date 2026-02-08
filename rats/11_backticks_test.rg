use "test"
use "str"

rats "backtick captures command output"
  result = test.run("rugo run rats/fixtures/backtick_simple.rg")
  test.assert_eq(result["status"], 0)
  expected = str.trim(`whoami`)
  test.assert_contains(result["output"], expected)
end

rats "backtick works with pipes"
  result = test.run("rugo run rats/fixtures/backtick_pipe.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "5")
end

rats "backtick with try/or for error handling"
  result = test.run("rugo run rats/fixtures/backtick_try.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "fallback")
end

rats "backtick result in expressions"
  result = test.run("rugo run rats/fixtures/backtick_expr.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello world")
end
