# RATS: Test control flow (if/elsif/else/while)
import "test"

rats "if statement true branch"
  test.run("printf 'if true\n  puts(\"yes\")\nend\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "yes")
end

rats "if/else false branch"
  test.run("printf 'if false\n  puts(\"yes\")\nelse\n  puts(\"no\")\nend\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "no")
end

rats "if/elsif/else chain"
  test.run("printf 'x = 2\nif x == 1\n  puts(\"one\")\nelsif x == 2\n  puts(\"two\")\nelse\n  puts(\"other\")\nend\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "two")
end

rats "while loop"
  test.run("printf 'i = 0\nwhile i < 3\n  puts(i)\n  i = i + 1\nend\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "0")
  test.assert_eq(lines[1], "1")
  test.assert_eq(lines[2], "2")
end
