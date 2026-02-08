import "test"

rats "bench keyword compiles and runs"
  result = test.run("rugo run rats/fixtures/bench_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "simple addition")
  test.assert_contains(result["output"], "/op")
  test.assert_contains(result["output"], "runs")
end

rats "bench with multiple blocks"
  result = test.run("rugo run rats/fixtures/bench_multi.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "fib(10)")
  test.assert_contains(result["output"], "fib(15)")
end

rats "bench with user-defined functions"
  result = test.run("rugo run rats/fixtures/bench_with_func.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "calls helper")
  test.assert_contains(result["output"], "/op")
end

rats "bench keyword in emit output"
  result = test.run("rugo emit rats/fixtures/bench_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugo_bench_runner")
  test.assert_contains(result["output"], "rugoBenchCase")
  test.assert_contains(result["output"], "rugo_bench_0")
end
