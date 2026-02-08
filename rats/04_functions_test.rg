# RATS: Test functions
import "test"

rats "user defined function"
  script = <<~SCRIPT
    def greet(name)
      puts("hi " + name)
    end
    greet("rugo")
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hi rugo")
end

rats "function with return value"
  script = <<~SCRIPT
    def double(x)
      return x * 2
    end
    puts(double(21))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "function calling another function"
  script = <<~SCRIPT
    def add(a, b)
      return a + b
    end
    def add3(a, b, c)
      return add(add(a, b), c)
    end
    puts(add3(1, 2, 3))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "6")
end

rats "paren-free function calls"
  result = test.run("rugo run examples/paren_free.rg")
  test.assert_eq(result["status"], 0)
end
