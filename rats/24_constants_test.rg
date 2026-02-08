# RATS: Constants (uppercase identifiers are immutable)
use "test"

rats "constants basic assignment and use"
  result = test.run("rugo run rats/fixtures/constants_basic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "100")
  test.assert_eq(lines[1], "3.14")
  test.assert_eq(lines[2], "Rugo")
end

rats "lowercase variables can be reassigned"
  result = test.run("rugo run rats/fixtures/constants_lowercase_ok.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "20")
end

rats "constants inside functions"
  result = test.run("rugo run rats/fixtures/constants_in_func.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "314.159")
end

rats "constants are scoped per function"
  result = test.run("rugo run rats/fixtures/constants_scope.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_eq(lines[1], "100")
end

rats "constant hash allows content mutation"
  result = test.run("rugo run rats/fixtures/constants_hash_mutation.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "localhost")
  test.assert_eq(lines[1], "30")
end

# --- Error cases ---

rats "reassigning constant is a compile error"
  result = test.run("rugo run rats/fixtures/err_constant_reassign.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot reassign constant MAX")
end

rats "reassigning constant in inner scope is a compile error"
  result = test.run("rugo run rats/fixtures/err_constant_reassign_inner.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot reassign constant Name")
end

rats "reassigning constant inside function is a compile error"
  result = test.run("rugo run rats/fixtures/err_constant_reassign_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot reassign constant Limit")
end
