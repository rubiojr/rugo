# RATS: Test subdirectory require auto-namespace (bug b2ec60e)
# require "dir/file" should use "file" as namespace, not "dir/file"
use "test"

rats "subdirectory require uses basename as namespace"
  result = test.run("rugo run rats/fixtures/subdir_require_auto_ns.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello world")
end

rats "subdirectory require with alias still works"
  result = test.run("rugo run rats/fixtures/subdir_require_alias.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello aliased")
end
