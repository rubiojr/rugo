# RATS: Test arrays and hashes
import "test"

rats "array creation and access"
  test.run("printf 'arr = [1, 2, 3]\nputs(arr[0])\nputs(arr[2])\nputs(len(arr))\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "1")
  test.assert_eq(lines[1], "3")
  test.assert_eq(lines[2], "3")
end

rats "array append"
  test.run("printf 'arr = [1, 2]\narr = append(arr, 3)\nputs(len(arr))\nputs(arr[2])\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "3")
  test.assert_eq(lines[1], "3")
end

rats "hash creation and access"
  test.run("printf 'h = {\"a\" => 1, \"b\" => 2}\nputs(h[\"a\"])\nputs(h[\"b\"])\n' > " + test.tmpdir() + "/test.rg")
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "1")
  test.assert_eq(lines[1], "2")
end

rats "arrays and hashes example"
  result = test.run("rugo run examples/arrays_hashes.rg")
  test.assert_eq(result["status"], 0)
end

rats "array slice basic"
  result = test.run("rugo run rats/fixtures/slice_basic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "3")
  test.assert_eq(lines[1], "1")
  test.assert_eq(lines[2], "3")
  test.assert_eq(lines[3], "4")
  test.assert_eq(lines[4], "4")
  test.assert_eq(lines[5], "2")
  test.assert_eq(lines[6], "0")
end
