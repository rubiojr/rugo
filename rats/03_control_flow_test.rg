# RATS: Test control flow (if/elsif/else/while)
use "test"

rats "if statement true branch"
  script = <<~SCRIPT
    if true
      puts("yes")
    end
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "yes")
end

rats "if/else false branch"
  script = <<~SCRIPT
    if false
      puts("yes")
    else
      puts("no")
    end
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "no")
end

rats "if/elsif/else chain"
  script = <<~SCRIPT
    x = 2
    if x == 1
      puts("one")
    elsif x == 2
      puts("two")
    else
      puts("other")
    end
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "two")
end

rats "while loop"
  script = <<~SCRIPT
    i = 0
    while i < 3
      puts(i)
      i = i + 1
    end
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "0")
  test.assert_eq(lines[1], "1")
  test.assert_eq(lines[2], "2")
end
