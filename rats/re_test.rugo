use "test"
use "re"

rats "re.test matches pattern"
  test.assert_true(re.test("^\\d+$", "42"))
  test.assert_false(re.test("^\\d+$", "abc"))
  test.assert_true(re.test("hello", "say hello world"))
end

rats "re.find returns first match"
  test.assert_eq(re.find("\\d+", "abc123def456"), "123")
end

rats "re.find returns nil on no match"
  test.assert_nil(re.find("\\d+", "no numbers"))
end

rats "re.find_all returns all matches"
  result = re.find_all("\\d+", "a1b2c3")
  test.assert_eq(len(result), 3)
  test.assert_eq(result[0], "1")
  test.assert_eq(result[1], "2")
  test.assert_eq(result[2], "3")
end

rats "re.find_all empty on no match"
  result = re.find_all("\\d+", "no numbers")
  test.assert_eq(len(result), 0)
end

rats "re.replace replaces first match"
  test.assert_eq(re.replace("\\d+", "a1b2c3", "X"), "aXb2c3")
end

rats "re.replace no match returns original"
  test.assert_eq(re.replace("\\d+", "hello", "X"), "hello")
end

rats "re.replace_all replaces all matches"
  test.assert_eq(re.replace_all("\\d+", "a1b2c3", "X"), "aXbXcX")
end

rats "re.split splits by pattern"
  result = re.split("\\s+", "hello   world   foo")
  test.assert_eq(len(result), 3)
  test.assert_eq(result[0], "hello")
  test.assert_eq(result[1], "world")
  test.assert_eq(result[2], "foo")
end

rats "re.split with comma separator"
  result = re.split(",\\s*", "a, b, c")
  test.assert_eq(len(result), 3)
  test.assert_eq(result[0], "a")
  test.assert_eq(result[1], "b")
  test.assert_eq(result[2], "c")
end

rats "re.match returns hash with groups"
  m = re.match("(\\w+)@(\\w+)", "contact: foo@bar.com")
  test.assert_eq(m["match"], "foo@bar")
  groups = m["groups"]
  test.assert_eq(len(groups), 2)
  test.assert_eq(groups[0], "foo")
  test.assert_eq(groups[1], "bar")
end

rats "re.match returns nil on no match"
  test.assert_nil(re.match("\\d+", "no numbers"))
end

rats "re.match without capture groups"
  m = re.match("hello", "say hello world")
  test.assert_eq(m["match"], "hello")
  test.assert_eq(len(m["groups"]), 0)
end

rats "re.test with special regex chars"
  test.assert_true(re.test("\\.", "hello.world"))
  test.assert_false(re.test("^\\.$", "hello"))
  test.assert_true(re.test("[a-z]+", "hello"))
  test.assert_true(re.test("^(foo|bar)$", "foo"))
end

rats "re invalid pattern panics"
  msg = try re.test("[invalid", "test") or "bad pattern"
  test.assert_eq(msg, "bad pattern")
end

rats "re.replace_all with backreference"
  result = re.replace_all("(\\w+)@(\\w+)", "foo@bar baz@qux", "$1 at $2")
  test.assert_eq(result, "foo at bar baz at qux")
end
