import "test"
import "str"

rats "error shows script filename and line for index out of range"
  result = test.run("rugo run rats/fixtures/error_index.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error:")
  test.assert_contains(result["output"], "error_index.rg:2")
end

rats "error shows correct line for type errors"
  result = test.run("rugo run rats/fixtures/error_type.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error_type.rg:3")
  test.assert_contains(result["output"], "cannot add")
end

rats "error inside function shows function body line"
  result = test.run("rugo run rats/fixtures/error_in_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error_in_func.rg:2")
  test.assert_contains(result["output"], "divide by zero")
end

rats "error lines account for comments"
  result = test.run("rugo run rats/fixtures/error_with_comments.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error_with_comments.rg:6")
end

rats "no panic stacktrace in error output"
  result = test.run("rugo run rats/fixtures/error_index.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error:")
  # Must NOT contain Go panic artifacts
  test.assert_false(str.contains(result["output"], "goroutine"))
  test.assert_false(str.contains(result["output"], "panic:"))
  test.assert_false(str.contains(result["output"], "main.go"))
end

rats "successful scripts have clean output"
  result = test.run("rugo run rats/fixtures/error_none.rg")
  test.assert_eq(result["status"], 0)
  test.assert_false(str.contains(result["output"], "error:"))
end

rats "built binary shows script line in errors"
  result = test.run("rugo build -o " + test.tmpdir() + "/errbin rats/fixtures/error_index.rg")
  test.assert_eq(result["status"], 0)
  result = test.run(test.tmpdir() + "/errbin")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error:")
  test.assert_contains(result["output"], "error_index.rg:2")
  test.assert_false(str.contains(result["output"], "goroutine"))
end
