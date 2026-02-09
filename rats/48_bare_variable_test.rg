# RATS: Regression test for bare variable not treated as shell command
# Covers: def, fn, spawn, for-loop variables, and mid-body bare variable
use "test"

# --- Positive: bare variable in various block types ---

rats "bare variable in def is not a shell command"
  result = test.run("rugo run rats/fixtures/bare_var_def.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Rugo")
end

rats "bare variable in fn is implicit return"
  result = test.run("rugo run rats/fixtures/bare_var_fn.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Rugo")
end

rats "bare variable in spawn is task result"
  result = test.run("rugo run rats/fixtures/bare_var_spawn.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello")
end

rats "bare variable mid-body in def is not a shell command"
  result = test.run("rugo run rats/fixtures/bare_var_mid_body.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "for-loop variable is not a shell command"
  result = test.run("rugo run rats/fixtures/bare_var_for.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "60")
end

# --- Edge: fn parameter as bare identifier ---

rats "fn parameter as bare implicit return"
  result = test.run("rugo run rats/fixtures/bare_var_fn_param.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

# --- Edge: for k, v in hash tracks both variables ---

rats "for k, v loop variables are not shell commands"
  result = test.run("rugo run rats/fixtures/bare_var_for_kv.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "2")
end

# --- Edge: bare variable inside if body ---

rats "bare variable in if body is not a shell command"
  result = test.run("rugo run rats/fixtures/bare_var_in_if.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

# --- Edge: variable shadows a shell command name ---

rats "variable shadowing shell command is treated as variable"
  result = test.run("rugo run rats/fixtures/bare_var_shadow_cmd.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "not a directory listing")
end

# --- Negative: shell commands still work ---

rats "shell fallback still works for unknown identifiers"
  result = test.run("rugo run examples/shell_fallback.rg")
  test.assert_eq(result["status"], 0)
end

# --- Build: bare variable compiles to native binary ---

rats "bare variable compiles to native binary"
  result = test.run("rugo build rats/fixtures/bare_var_build.rg -o /tmp/bare_var_build_test")
  test.assert_eq(result["status"], 0)
  result = test.run("/tmp/bare_var_build_test")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Rugo")
  test.run("rm -f /tmp/bare_var_build_test")
end
