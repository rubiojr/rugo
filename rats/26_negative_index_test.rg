# RATS: Test negative array indexing
import "test"

rats "negative index reads from end"
  result = test.run("rugo run rats/fixtures/negative_index.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "50")
  test.assert_eq(lines[1], "40")
  test.assert_eq(lines[2], "10")
end

rats "negative index assignment"
  result = test.run("rugo run rats/fixtures/negative_index_assign.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "99")
end

rats "negative index -1 returns last element"
  script = <<~SCRIPT
    arr = ["a", "b", "c"]
    puts(arr[-1])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "c")
end

rats "positive indices still work"
  script = <<~SCRIPT
    arr = [10, 20, 30]
    puts(arr[0])
    puts(arr[2])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "10")
  test.assert_eq(lines[1], "30")
end

rats "hash with negative integer key still works"
  script = <<~SCRIPT
    h = {-1 => "neg"}
    puts(h[-1])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "neg")
end
