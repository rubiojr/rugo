# Test: optional parentheses for zero-argument function definitions
use "test"

# --- Positive tests ---

rats "def without parens - basic"
  result = test.run("rugo run rats/fixtures/def_no_parens_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello from greet")
end

rats "def with empty parens still works"
  result = test.run("rugo run rats/fixtures/def_empty_parens.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello from parens")
end

rats "def with params still works"
  result = test.run("rugo run rats/fixtures/def_with_params.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello Alice")
end

rats "def no parens with return value"
  result = test.run("rugo run rats/fixtures/def_no_parens_return.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "42")
end

rats "multiple def styles mixed"
  result = test.run("rugo run rats/fixtures/def_mixed_styles.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "no-parens")
  test.assert_contains(result["output"], "empty-parens")
  test.assert_contains(result["output"], "with-params: hello")
end

rats "def no parens called paren-free"
  result = test.run("rugo run rats/fixtures/def_no_parens_call.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "working")
end

rats "def no parens in require'd file"
  result = test.run("rugo run rats/fixtures/def_no_parens_require.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "from helper")
end

rats "def no parens with if/while/for inside"
  result = test.run("rugo run rats/fixtures/def_no_parens_blocks.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "positive")
  test.assert_contains(result["output"], "counted to 3")
end

# --- Negative tests ---

rats "def without end still errors"
  result = test.run("rugo run rats/fixtures/err_def_no_end.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "unclosed")
end

rats "def with bad param list still errors"
  result = test.run("rugo run rats/fixtures/err_def_bad_params.rg")
  test.assert_neq(result["status"], 0)
end

rats "def empty name is an error"
  result = test.run("rugo run rats/fixtures/err_def_no_name.rg")
  test.assert_neq(result["status"], 0)
end
