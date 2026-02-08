# RATS: Test argument count validation
import "test"

rats "too many arguments"
  result = test.run("rugo run rats/fixtures/err_too_many_args.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "wrong number of arguments for greet (2 for 1)")
end

rats "too few arguments"
  result = test.run("rugo run rats/fixtures/err_too_few_args.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "wrong number of arguments for add (1 for 2)")
end

rats "zero arguments when one expected"
  result = test.run("rugo run rats/fixtures/err_zero_args.rg")
  test.assert_eq(result["status"], 1)
  test.assert_contains(result["output"], "wrong number of arguments for greet (0 for 1)")
end

rats "error message does not expose rugofn_ prefix"
  result = test.run("rugo run rats/fixtures/err_too_many_args.rg")
  test.assert_eq(result["status"], 1)
  # The error should mention the clean name, not the internal prefix
  test.assert_contains(result["output"], "wrong number of arguments for greet")
end

rats "correct argument count works"
  script = <<~SCRIPT
    def add(a, b)
      return a + b
    end
    puts(add(1, 2))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "3")
end

rats "zero-param function works with no args"
  script = <<~SCRIPT
    def hello()
      puts("hi")
    end
    hello()
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hi")
end
