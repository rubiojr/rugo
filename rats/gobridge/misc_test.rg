# RATS: Test Go bridge — regexp, sort, time, os packages
use "test"

# regexp
rats "regexp.match_string"
  script = <<~SCRIPT
    import "regexp"
    puts(regexp.match_string("^hello", "hello world"))
    puts(regexp.match_string("^world", "hello world"))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "true")
  test.assert_contains(result["output"], "false")
end

rats "regexp.match_string basic"
  script = <<~SCRIPT
    import "regexp"
    puts(regexp.match_string("[a-z]+", "hello"))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "true")
end

# sort
rats "sort.strings"
  script = <<~SCRIPT
    import "sort"
    arr = ["banana", "apple", "cherry"]
    sorted = sort.strings(arr)
    puts(sorted[0])
    puts(sorted[1])
    puts(sorted[2])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "apple")
end

rats "sort.ints"
  script = <<~SCRIPT
    import "sort"
    use "conv"
    arr = [3, 1, 2]
    sorted = sort.ints(arr)
    puts(conv.to_s(sorted[0]))
    puts(conv.to_s(sorted[1]))
    puts(conv.to_s(sorted[2]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "1")
end

rats "sort.is_sorted"
  script = <<~SCRIPT
    import "sort"
    puts(sort.is_sorted(["apple", "banana", "cherry"]))
    puts(sort.is_sorted(["cherry", "apple"]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "true")
  test.assert_contains(result["output"], "false")
end

# time
rats "time.now_unix"
  script = <<~SCRIPT
    import "time"
    use "conv"
    ts = time.now_unix()
    puts(conv.to_s(ts > 0))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "true")
end

rats "time.sleep_ms"
  script = <<~SCRIPT
    import "time"
    use "conv"
    start = time.now_unix_nano()
    time.sleep_ms(50)
    elapsed = time.now_unix_nano() - start
    puts(conv.to_s(elapsed > 40000000))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "true")
end

# os (Go bridge with alias)
rats "os bridge getenv with alias"
  script = <<~SCRIPT
    import "os" as go_os
    home = go_os.getenv("HOME")
    puts(home)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_neq(result["output"], "")
end

rats "os bridge getwd"
  script = <<~SCRIPT
    import "os" as go_os
    cwd = go_os.getwd()
    puts(cwd)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_neq(result["output"], "")
end

rats "os bridge hostname"
  script = <<~SCRIPT
    import "os" as go_os
    h = go_os.hostname()
    puts(h)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_neq(result["output"], "")
end

rats "os bridge temp_dir"
  script = <<~SCRIPT
    import "os" as go_os
    puts(go_os.temp_dir())
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "tmp")
end

rats "os bridge read_file"
  script = <<~SCRIPT
    import "os" as go_os
    go_os.mkdir_all("/tmp/rugo_test_bridge", 493)
    content = go_os.read_file("/tmp/rugo_test_bridge/../rugo_test_bridge/../../etc/os-release")
    puts(content)
    go_os.remove_all("/tmp/rugo_test_bridge")
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
end
