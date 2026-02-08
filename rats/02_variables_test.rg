# RATS: Test variables, types, and expressions
use "test"

rats "variables and arithmetic"
  script = <<~SCRIPT
    x = 10
    y = 20
    puts(x + y)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "30")
end

rats "string concatenation"
  script = <<~SCRIPT
    name = "world"
    puts("hello " + name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello world")
end

rats "boolean values"
  script = <<~SCRIPT
    puts(true)
    puts(false)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "true")
  test.assert_eq(lines[1], "false")
end

rats "nil value"
  script = <<~SCRIPT
    x = nil
    puts(x)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "<nil>")
end

rats "string interpolation"
  script = <<~SCRIPT
    name = "rugo"
    puts("hello " + name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello rugo")
end

rats "arithmetic operations"
  script = <<~SCRIPT
    puts(2 + 3)
    puts(10 - 4)
    puts(3 * 7)
    puts(15 / 3)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "5")
  test.assert_eq(lines[1], "6")
  test.assert_eq(lines[2], "21")
  test.assert_eq(lines[3], "5")
end

rats "comparison operators"
  script = <<~SCRIPT
    puts(1 == 1)
    puts(1 != 2)
    puts(3 > 2)
    puts(2 < 3)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "true")
  test.assert_eq(lines[1], "true")
  test.assert_eq(lines[2], "true")
  test.assert_eq(lines[3], "true")
end

rats "logical operators"
  script = <<~SCRIPT
    puts(true && true)
    puts(true && false)
    puts(false || true)
    puts(false || false)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "true")
  test.assert_eq(lines[1], "false")
  test.assert_eq(lines[2], "true")
  test.assert_eq(lines[3], "false")
end
