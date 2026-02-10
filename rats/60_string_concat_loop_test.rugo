# RATS: String concatenation with dynamic variables in loops
#
# Regression test for: variable + "literal" inside loops where
# the variable gets widened to interface{} (e.g. from hash iteration).
# The codegen must use rugo_add() instead of raw Go + in these cases.
use "test"
use "conv"

rats "string concat: variable + literal in for-hash loop"
  commands = {"help" => "List commands", "ping" => "Pong"}
  lines = "cmds:\n"
  for name, desc in commands
    lines = lines + "- " + name + "\n"
  end
  test.assert_contains(lines, "cmds:")
  test.assert_contains(lines, "- help")
  test.assert_contains(lines, "- ping")
end

rats "string concat: variable + variable in for-hash loop"
  commands = {"help" => "List commands", "ping" => "Pong"}
  lines = ""
  for name, desc in commands
    lines = lines + name + ": " + desc + "\n"
  end
  test.assert_contains(lines, "help: List commands")
  test.assert_contains(lines, "ping: Pong")
end

rats "string concat: literal + variable in for-hash loop"
  items = {"a" => "1", "b" => "2"}
  result = ""
  for k, v in items
    result = "(" + k + "=" + v + ") " + result
  end
  test.assert_contains(result, "(a=1)")
  test.assert_contains(result, "(b=2)")
end

rats "string concat: variable + literal in while loop with dynamic var"
  arr = [1, 2, 3]
  s = ""
  i = 0
  while i < len(arr)
    s = s + "item" + conv.to_s(arr[i]) + " "
    i = i + 1
  end
  test.assert_contains(s, "item1")
  test.assert_contains(s, "item2")
  test.assert_contains(s, "item3")
end

rats "int arithmetic in for loop is not regressed"
  sum = 0
  for i in [1, 2, 3, 4, 5]
    sum = sum + i
  end
  test.assert_eq(sum, 15)
end

rats "pure string concat in for loop still works"
  s = ""
  for i in [1, 2, 3]
    s = s + "x"
  end
  test.assert_eq(s, "xxx")
end
