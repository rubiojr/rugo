# RATS: Test require'd file scope — sibling calls and constants (bug 165941f)
use "test"

rats "sibling function call and constant access in require'd file"
  result = test.run("rugo run rats/fixtures/require_scope_main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "hello from https://api.example.com")
end

rats "sibling function call emits rugons_ prefix"
  result = test.run("rugo emit rats/fixtures/require_scope_main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugons_require_scope_lib_helper()")
end

rats "file-level constant emitted as package-level var"
  result = test.run("rugo emit rats/fixtures/require_scope_main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "var BASE_URL interface{}")
end
