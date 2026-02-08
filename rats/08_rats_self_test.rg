# RATS: Test the RATS framework itself
use "test"

rats "test.assert_eq passes on equal values"
  test.assert_eq(1, 1)
  test.assert_eq("hello", "hello")
  test.assert_eq(true, true)
end

rats "test.assert_neq passes on different values"
  test.assert_neq(1, 2)
  test.assert_neq("a", "b")
  test.assert_neq(true, false)
end

rats "test.assert_true passes on truthy"
  test.assert_true(true)
  test.assert_true(1)
  test.assert_true("notempty")
end

rats "test.assert_false passes on falsy"
  test.assert_false(false)
  test.assert_false(nil)
end

rats "test.assert_contains works"
  test.assert_contains("hello world", "world")
  test.assert_contains("foobar", "bar")
end

rats "test.assert_nil passes on nil"
  test.assert_nil(nil)
end

rats "test.run returns status and output"
  result = test.run("echo ok")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "ok")
end

rats "test.run returns lines array"
  result = test.run("printf 'line1\nline2\nline3'")
  lines = result["lines"]
  test.assert_eq(len(lines), 3)
  test.assert_eq(lines[0], "line1")
  test.assert_eq(lines[1], "line2")
  test.assert_eq(lines[2], "line3")
end

rats "test.run captures nonzero exit"
  result = test.run("exit 1")
  test.assert_eq(result["status"], 1)
end

rats "test.skip skips the test"
  test.skip("testing skip")
  test.fail("should not reach here")
end
