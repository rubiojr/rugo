# RATS: Test spawn concurrency
# Covers spawn block, one-liner, fan-out, task methods,
# error propagation, timeouts, and syntax errors.
use "test"
use "str"

# --- Positive: spawn block with .value ---

rats "spawn block returns value via .value"
  result = test.run("rugo run rats/fixtures/spawn_value.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "spawn fire-and-forget runs in background"
  result = test.run("rugo run rats/fixtures/spawn_fire_forget.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "bg")
  test.assert_contains(result["output"], "main")
end

rats "spawn one-liner sugar"
  result = test.run("rugo run rats/fixtures/spawn_oneliner.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "spawn multi-line body"
  result = test.run("rugo run rats/fixtures/spawn_multiline.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "30")
end

# --- Positive: parallel fan-out ---

rats "spawn fan-out collects ordered results"
  result = test.run("rugo run rats/fixtures/spawn_fanout.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "10")
  test.assert_eq(lines[1], "20")
  test.assert_eq(lines[2], "30")
end

# --- Positive: error handling with try/or ---

rats "spawn error caught with try/or default"
  result = test.run("rugo run rats/fixtures/spawn_try_or.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "caught")
end

rats "spawn error caught with try/or handler block"
  result = test.run("rugo run rats/fixtures/spawn_try_handler.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "err:")
end

# --- Positive: task.done polling ---

rats "task.done returns true after completion"
  result = test.run("rugo run rats/fixtures/spawn_done.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "finished")
end

# --- Positive: task.wait with timeout ---

rats "task.wait succeeds when task completes in time"
  result = test.run("rugo run rats/fixtures/spawn_wait_ok.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello")
end

rats "task.wait times out and caught with try/or"
  result = test.run("rugo run rats/fixtures/spawn_timeout.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "timed out")
end

# --- Positive: spawn inside function ---

rats "spawn returned from a function"
  result = test.run("rugo run rats/fixtures/spawn_in_func.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "500")
end

# --- Positive: empty body ---

rats "spawn with empty body does not crash"
  result = test.run("rugo run rats/fixtures/spawn_empty_body.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "done")
end

# --- Positive: no spawn means no spawn runtime ---

rats "files without spawn omit spawn runtime"
  result = test.run("rugo emit rats/fixtures/spawn_no_spawn.rg")
  test.assert_eq(result["status"], 0)
  test.assert_false(str.contains(result["output"], "rugoTask"))
  test.assert_false(str.contains(result["output"], "rugo_task_value"))
end

rats "files with spawn include spawn runtime"
  result = test.run("rugo emit rats/fixtures/spawn_value.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugoTask")
  test.assert_contains(result["output"], "rugo_task_value")
  test.assert_contains(result["output"], "rugo_task_done")
  test.assert_contains(result["output"], "rugo_task_wait")
end

# --- Positive: build to native binary ---

rats "spawn compiles to native binary"
  result = test.run("rugo build -o " + test.tmpdir() + "/spawn_bin rats/fixtures/spawn_value.rg")
  test.assert_eq(result["status"], 0)
  result = test.run(test.tmpdir() + "/spawn_bin")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

# --- Negative: syntax errors ---

rats "spawn missing end is a parse error"
  result = test.run("rugo run rats/fixtures/err_spawn_missing_end.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "error:")
  test.assert_contains(result["output"], "end")
end

# --- Negative: .value on non-task ---

rats ".value on non-task is a runtime error"
  result = test.run("rugo run rats/fixtures/err_spawn_value_non_task.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot call .value on int")
  test.assert_contains(result["output"], "expected a spawn task")
  test.assert_contains(result["output"], "err_spawn_value_non_task.rg:2")
end

# --- Negative: .done on non-task ---

rats ".done on non-task is a runtime error"
  result = test.run("rugo run rats/fixtures/err_spawn_done_non_task.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot call .done on int")
  test.assert_contains(result["output"], "expected a spawn task")
  test.assert_contains(result["output"], "err_spawn_done_non_task.rg:2")
end

# --- Negative: .wait on non-task ---

rats ".wait on non-task is a runtime error"
  result = test.run("rugo run rats/fixtures/err_spawn_wait_non_task.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot call .wait on int")
  test.assert_contains(result["output"], "expected a spawn task")
  test.assert_contains(result["output"], "err_spawn_wait_non_task.rg:2")
end

# --- Negative: panic inside spawn propagates on .value ---

rats "panic in spawn propagates through .value"
  result = test.run("rugo run rats/fixtures/err_spawn_panic_value.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "division by zero")
end

# --- Negative: error output format ---

rats "spawn runtime errors show .rg file and line"
  result = test.run("rugo run rats/fixtures/err_spawn_panic_value.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "err_spawn_panic_value.rg:")
  # No Go stack traces leak
  test.assert_false(str.contains(result["output"], "goroutine"))
  test.assert_false(str.contains(result["output"], "panic:"))
end

# --- Regression: spawn can reassign outer typed variable ---

rats "spawn reassigns outer typed variable"
  result = test.run("rugo run rats/fixtures/spawn_outer_var.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "after")
end
