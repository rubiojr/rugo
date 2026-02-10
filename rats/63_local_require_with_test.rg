# RATS: Test local require with 'with' clause
# require "dir" with mod1, mod2 loads specific .rg files from a local directory
use "test"

rats "local require with loads multiple modules"
  result = test.run("rugo run rats/fixtures/local_with_basic.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "hello, world")
  test.assert_eq(lines[1], "42")
end

rats "local require with loads a single module"
  result = test.run("rugo run rats/fixtures/local_with_single.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello, single")
end

rats "local require with propagates use statements"
  result = test.run("rugo run rats/fixtures/local_with_use_propagation.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "value=10")
end

rats "local require with works with subdirectory path"
  script = <<~SCRIPT
    require "local_with_lib" with math
    puts(math.triple(7))
  SCRIPT
  test.write_file(test.tmpdir() + "/subdir_with.rg", script)
  # Copy the library dir to tmpdir so relative path works
  test.run("cp -r rats/fixtures/local_with_lib " + test.tmpdir() + "/local_with_lib")
  result = test.run("rugo run " + test.tmpdir() + "/subdir_with.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "21")
end

rats "local require with build produces working binary"
  tmpdir = test.tmpdir()
  test.run("cp -r rats/fixtures/local_with_lib " + tmpdir + "/local_with_lib")
  script = <<~SCRIPT
    require "local_with_lib" with greet, math
    puts(greet.greet("binary"))
    puts(math.double(10))
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  build_result = test.run("rugo build " + tmpdir + "/main.rg -o " + tmpdir + "/main_bin")
  test.assert_eq(build_result["status"], 0)
  result = test.run(tmpdir + "/main_bin")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "hello, binary")
  test.assert_eq(lines[1], "20")
end

# --- Negative tests ---

rats "local require with fails when path is not a directory"
  script = <<~SCRIPT
    require "local_with_basic.rg" with greet
    puts(greet.greet("x"))
  SCRIPT
  test.write_file(test.tmpdir() + "/err_file.rg", script)
  test.run("touch " + test.tmpdir() + "/local_with_basic.rg")
  result = test.run("rugo run " + test.tmpdir() + "/err_file.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "not a directory")
end

rats "local require with fails when module file is missing"
  tmpdir = test.tmpdir()
  test.run("cp -r rats/fixtures/local_with_lib " + tmpdir + "/local_with_lib")
  script = <<~SCRIPT
    require "local_with_lib" with nonexistent
  SCRIPT
  test.write_file(tmpdir + "/err_missing.rg", script)
  result = test.run("rugo run " + tmpdir + "/err_missing.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "nonexistent")
  test.assert_contains(result["output"], "not found")
end

rats "local require with fails when path does not exist"
  script = <<~SCRIPT
    require "no_such_dir" with foo
  SCRIPT
  test.write_file(test.tmpdir() + "/err_nodir.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/err_nodir.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "not a directory")
end

rats "local require with detects namespace conflict with use"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mylib")
  test.write_file(tmpdir + "/mylib/os.rg", "def info()\n  return \"hi\"\nend\n")
  script = <<~SCRIPT
    use "os"
    require "mylib" with os
  SCRIPT
  test.write_file(tmpdir + "/err_ns.rg", script)
  result = test.run("rugo run " + tmpdir + "/err_ns.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "conflicts")
end

rats "local require with falls back to lib/ subdirectory"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mylib/lib")
  test.write_file(tmpdir + "/mylib/lib/utils.rg", "def hello()\n  return \"from lib\"\nend\n")
  script = <<~SCRIPT
    require "mylib" with utils
    puts(utils.hello())
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "from lib")
end

rats "local require with prefers root over lib/"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mylib/lib")
  test.write_file(tmpdir + "/mylib/utils.rg", "def hello()\n  return \"from root\"\nend\n")
  test.write_file(tmpdir + "/mylib/lib/utils.rg", "def hello()\n  return \"from lib\"\nend\n")
  script = <<~SCRIPT
    require "mylib" with utils
    puts(utils.hello())
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "from root")
end
