use "test"
use "str"

# --- Positive tests ---

rats "fn dot-call: basic lambda via dot access"
  result = test.run("rugo run rats/fixtures/fn_dot_call_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "hello test")
end

rats "fn dot-call: lambda with arguments"
  result = test.run("rugo run rats/fixtures/fn_dot_call_args.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "Bob")
end

rats "fn dot-call: colon syntax hash with lambdas"
  result = test.run("rugo run rats/fixtures/fn_dot_call_colon.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "5")
  test.assert_eq(result["lines"][1], "20")
end

rats "fn dot-call: zero-arg lambda"
  result = test.run("rugo run rats/fixtures/fn_dot_call_zero_args.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "pong")
end

rats "fn dot-call: multi-arg lambda"
  result = test.run("rugo run rats/fixtures/fn_dot_call_multi_args.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "5")
  test.assert_eq(result["lines"][1], "0")
  test.assert_eq(result["lines"][2], "10")
end

rats "fn dot-call: both index and dot access work"
  result = test.run("rugo run rats/fixtures/fn_dot_call_both_access.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "5")
  test.assert_eq(result["lines"][1], "5")
end

rats "fn dot-call: factory pattern with closure"
  result = test.run("rugo run rats/fixtures/fn_dot_call_factory.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "Hello, Alice")
end

rats "fn dot-call: lambda mutates hash state"
  result = test.run("rugo run rats/fixtures/fn_dot_call_mutate.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "false")
  test.assert_eq(result["lines"][1], "saved Alice")
  test.assert_eq(result["lines"][2], "true")
end

# --- Negative tests ---

rats "fn dot-call: error on missing key"
  result = test.run("rugo run rats/fixtures/fn_dot_call_missing.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "undefined method .missing_method()")
  test.assert_contains(result["output"], "not found in hash")
end

rats "fn dot-call: error calling non-function value"
  result = test.run("rugo run rats/fixtures/fn_dot_call_not_fn.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "cannot call .name()")
  test.assert_contains(result["output"], "not a function")
end

rats "fn dot-call: compiles to native binary"
  result = test.run("rugo build rats/fixtures/fn_dot_call_basic.rg -o /tmp/fn_dot_call_test")
  test.assert_eq(result["status"], 0)
  result = test.run("/tmp/fn_dot_call_test")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["lines"][0], "hello test")
  test.run("rm -f /tmp/fn_dot_call_test")
end
