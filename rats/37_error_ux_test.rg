# Tests for UX improvements to error message formatting.

use "test"
use "str"

# --- Bug 4152d66: RATS runner "Error:" casing ---

rats "rats runner errors use lowercase error: prefix"
  result = test.run("rugo rats rats/fixtures/err_rats_parse.rg")
  test.assert_neq(result["status"], 0)
  # Must use lowercase "error:" not "Error:"
  test.assert_false(str.contains(result["output"], "Error:"))
  test.assert_contains(result["output"], "error:")
end

# --- Bug ba4701a: Relative paths in errors ---

rats "errors show relative path not absolute"
  result = test.run("rugo run rats/fixtures/err_rats_parse.rg")
  test.assert_neq(result["status"], 0)
  # Should NOT contain absolute path prefix like /home/
  test.assert_false(str.contains(result["output"], "/home/"))
  # Should contain the relative filename
  test.assert_contains(result["output"], "err_rats_parse.rg")
end

# --- Bug bcceb3e: Only show first parser error ---

rats "only first parser error shown"
  result = test.run("rugo run rats/fixtures/err_parse_cascade.rg")
  test.assert_neq(result["status"], 0)
  # Should show only one "expected" error line, not multiple
  lines = result["lines"]
  error_count = 0
  for line in lines
    if str.contains(line, "expected")
      error_count += 1
    end
  end
  test.assert_eq(error_count, 1)
end

# --- Bug 064c0c6 + 922de36: Human-friendly parser errors ---

rats "parser errors do not show grammar symbol names"
  result = test.run("rugo run rats/fixtures/err_rats_parse.rg")
  test.assert_neq(result["status"], 0)
  # Must NOT contain internal grammar names
  test.assert_false(str.contains(result["output"], "HashLit"))
  test.assert_false(str.contains(result["output"], "ParallelExpr"))
  test.assert_false(str.contains(result["output"], "AssignOrExpr"))
  test.assert_false(str.contains(result["output"], "ReturnStmt"))
  test.assert_false(str.contains(result["output"], "FuncDef"))
end

rats "parser errors use friendly token type names"
  result = test.run("rugo run rats/fixtures/err_missing_end.rg")
  test.assert_neq(result["status"], 0)
  # Should NOT show raw token type brackets like [EOF] or [str_lit]
  test.assert_false(str.contains(result["output"], "[EOF]"))
  test.assert_false(str.contains(result["output"], "[str_lit]"))
  test.assert_false(str.contains(result["output"], "[ident]"))
end

rats "missing paren shows clean error"
  result = test.run("rugo run rats/fixtures/err_missing_paren.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "expected")
  test.assert_contains(result["output"], ")")
  test.assert_false(str.contains(result["output"], "HashLit"))
end

# --- Bug 69766bf: Stray "end" keyword ---

rats "stray end keyword shows helpful error"
  result = test.run("rugo run rats/fixtures/err_stray_end.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "end")
  test.assert_contains(result["output"], "no matching")
  test.assert_false(str.contains(result["output"], "HashLit"))
end

# --- Bug 5946629: "or" without "try" hint ---

rats "or without try suggests try/or pattern"
  result = test.run("rugo run rats/fixtures/err_or_no_try.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "try")
  test.assert_contains(result["output"], "or")
  test.assert_false(str.contains(result["output"], "HashLit"))
end

# --- Bug becdec4: Duplicate function shows clean error ---

rats "duplicate function does not leak Go internals"
  result = test.run("rugo run rats/fixtures/err_duplicate_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "already defined")
  # Must NOT contain Go internal names
  test.assert_false(str.contains(result["output"], "rugofn_"))
  test.assert_false(str.contains(result["output"], "main.go"))
end

# --- Bug 606c5a5: Go build errors translated ---

rats "break outside loop uses Rugo terminology"
  result = test.run("rugo run rats/fixtures/err_break_outside.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "break")
  # Should NOT contain Go-specific "switch" or "select"
  test.assert_false(str.contains(result["output"], "switch"))
  test.assert_false(str.contains(result["output"], "select"))
  # Should NOT contain "# rugo_program" or "exit status"
  test.assert_false(str.contains(result["output"], "rugo_program"))
end

rats "next outside loop uses Rugo terminology"
  result = test.run("rugo run rats/fixtures/err_next_outside.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "next")
  # Should NOT say "continue" (Go term)
  test.assert_false(str.contains(result["output"], "continue"))
end

# --- Bug 18326af: Friendlier arg count format ---

rats "wrong arg count uses friendly format"
  result = test.run("rugo run rats/fixtures/err_too_many_args.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "takes")
  test.assert_contains(result["output"], "given")
  # Should NOT use old "(N for M)" format
  test.assert_false(str.contains(result["output"], " for "))
end

# --- Bug dcfd010: Simplified require error ---

rats "missing require shows clean error"
  result = test.run("rugo run rats/fixtures/err_require_missing.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "nonexistent_lib")
  # Should NOT show duplicate file paths or deeply nested error chains
  test.assert_false(str.contains(result["output"], "reading"))
  test.assert_false(str.contains(result["output"], "open"))
end

# --- Bug 54c3be5: Did-you-mean for module typos ---

rats "misspelled module suggests correct name"
  result = test.run("rugo run rats/fixtures/err_module_typo.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "did you mean")
  test.assert_contains(result["output"], "http")
end

# --- Bug 2861f85: Constant reassignment shows first assignment ---

rats "constant reassignment shows original location"
  result = test.run("rugo run rats/fixtures/err_constant_reassign.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot reassign constant MAX")
  test.assert_contains(result["output"], "line 1")
end

# --- Bug 75b716c: Go bridge shows snake_case names ---

rats "go bridge errors use snake_case names"
  result = test.run("rugo run rats/fixtures/err_bridge_name.rg")
  test.assert_neq(result["status"], 0)
  # Should use Rugo snake_case name, not Go PascalCase
  test.assert_contains(result["output"], "strconv.atoi")
  test.assert_false(str.contains(result["output"], "strconv.Atoi"))
end

# --- Bug a2ff7aa: Top-level error includes file name ---

rats "use inside function shows file and hint"
  result = test.run("rugo run rats/fixtures/err_use_in_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "use")
  test.assert_contains(result["output"], "top")
end

# --- Bug cc2843d: Cleaner regex errors ---

rats "regex error does not show Go regexp prefix"
  result = test.run("rugo run rats/fixtures/err_regex_bad.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "re.test")
  # Should NOT contain raw Go regexp error prefix
  test.assert_false(str.contains(result["output"], "error parsing regexp:"))
end

# --- Bug 5ae597f: Cleaner JSON errors ---

rats "json parse error is clean"
  result = test.run("rugo run rats/fixtures/err_json_parse.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "json.parse")
  test.assert_contains(result["output"], "invalid JSON")
end

# --- Bug 8746aa6: Cleaner HTTP errors ---

rats "http error does not show raw Go network errors"
  result = test.run("rugo run rats/fixtures/err_http_connect.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "http.get")
  # Should NOT show raw Go error chain like "dial tcp" or "Get "
  test.assert_false(str.contains(result["output"], "dial tcp"))
  test.assert_false(str.contains(result["output"], "Get \""))
end

# --- Bug 4f32a53: Conv module uses Rugo type names ---

rats "conv error shows Rugo type names"
  result = test.run("rugo run rats/fixtures/err_conv_type.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "array")
  test.assert_false(str.contains(result["output"], "[]interface"))
end

# --- Bug 962959e: Module arity errors ---

rats "module arity error shows given vs expected"
  result = test.run("rugo run rats/fixtures/err_module_arity.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "str.upper() takes 1 argument but 3 were given")
end

# --- Bug 31e5d62: Codegen errors include file:line ---

rats "codegen errors include file and line"
  result = test.run("rugo run rats/fixtures/err_duplicate_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "err_duplicate_func.rg:4:")

  result = test.run("rugo run rats/fixtures/err_use_in_func.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "err_use_in_func.rg:2:")
end

# --- Bug e416304: Namespace conflict errors include file:line ---

rats "namespace conflict errors include file and line"
  result = test.run("rugo run rats/fixtures/err_ns_conflict.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "err_ns_conflict.rg:2:")
  test.assert_contains(result["output"], "conflicts")
end

# --- Bug 4e3b071: Runtime panic messages translated ---

rats "runtime index out of bounds is friendly"
  result = test.run("rugo run rats/fixtures/err_index_oob.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "index 10 out of bounds")
  test.assert_false(str.contains(result["output"], "runtime error:"))
end

# --- Bug 212ed4a: Unclosed block names which block ---

rats "unclosed block names the block type"
  result = test.run("rugo run rats/fixtures/err_unclosed_def.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "unclosed \"def\" block")
  test.assert_contains(result["output"], "opened at line 1")
end

# --- Bug 4fed638: Unterminated delimiters show opening location ---

rats "unterminated string shows opening location"
  result = test.run("rugo run rats/fixtures/err_unterm_string.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "unterminated string literal")
  test.assert_contains(result["output"], "opened at line 1")
end

# --- Bug 6a3a0c2: Source code snippets in parse errors ---

rats "parse errors show source code snippet"
  result = test.run("rugo run rats/fixtures/err_missing_paren.rg")
  test.assert_neq(result["status"], 0)
  # Should contain the source line and a caret
  test.assert_contains(result["output"], "|")
  test.assert_contains(result["output"], "^")
end

# --- Bug 642836e: ANSI color in error messages ---

rats "error messages respect NO_COLOR"
  result = test.run("NO_COLOR=1 rugo run rats/fixtures/err_missing_paren.rg")
  test.assert_neq(result["status"], 0)
  test.assert_false(str.contains(result["output"], "\033["))
  test.assert_contains(result["output"], "error:")
end

rats "error messages have ANSI color when forced"
  result = test.run("unset NO_COLOR; RUGO_FORCE_COLOR=1 rugo run rats/fixtures/err_missing_paren.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "\033[")
end

# --- def without parens shows helpful hint ---

rats "def without parens explains the fix"
  result = test.run("rugo run rats/fixtures/err_def_no_parens.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "missing its parameter list")
  test.assert_contains(result["output"], "def greet")
end
