use "test"
use "str"

rats "exit with no args exits with code 0"
  result = test.run("rugo run rats/fixtures/exit_bare.rg")
  test.assert_eq(result["status"], 0)
end

rats "exit(0) exits with code 0"
  result = test.run("rugo run rats/fixtures/exit_zero.rg")
  test.assert_eq(result["status"], 0)
end

rats "exit(1) exits with code 1"
  result = test.run("rugo run rats/fixtures/exit_one.rg")
  test.assert_eq(result["status"], 1)
end

rats "exit(42) exits with code 42"
  result = test.run("rugo run rats/fixtures/exit_42.rg")
  test.assert_eq(result["status"], 42)
end

rats "exit stops execution"
  result = test.run("rugo run rats/fixtures/exit_stops.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "before")
  test.assert_eq(str.contains(result["output"], "SHOULD NOT PRINT"), false)
end

rats "exit with paren-free syntax"
  result = test.run("rugo run rats/fixtures/exit_paren_free.rg")
  test.assert_eq(result["status"], 5)
end

rats "exit inside a function"
  result = test.run("rugo run rats/fixtures/exit_in_func.rg")
  test.assert_eq(result["status"], 3)
  test.assert_eq(str.contains(result["output"], "SHOULD NOT PRINT"), false)
end
