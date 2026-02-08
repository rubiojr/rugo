# Tests for comparison operators including string ordering and numeric coercion.

import "test"

rats "string lexicographic less-than"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "lt_ok")
end

rats "string lexicographic greater-than"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "gt_ok")
end

rats "string greater-than-or-equal"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "gte_ok")
end

rats "string less-than-or-equal"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "lte_ok")
end

rats "string equality"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "eq_ok")
end

rats "string inequality"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "neq_ok")
end

rats "int == float numeric coercion"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "int_float_eq")
end

rats "int != float numeric coercion"
  result = test.run("rugo run rats/fixtures/string_compare.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "int_float_neq")
end
