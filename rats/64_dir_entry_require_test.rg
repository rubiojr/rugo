# RATS: Test require with directory-as-module (local entry point resolution)
# require "dir" resolves to dir/<dirname>.rg, dir/main.rg, or the sole .rg file
use "test"

rats "require directory resolves <dirname>.rg entry point"
  result = test.run("rugo run rats/fixtures/dir_entry_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello from dir_entry_lib, world")
end

rats "require directory resolves main.rg fallback"
  result = test.run("rugo run rats/fixtures/dir_entry_main_test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hi from main.rg")
end

rats "require directory with alias works"
  result = test.run("rugo run rats/fixtures/dir_entry_alias.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello from dir_entry_lib, alias")
end

rats "require directory resolves sole .rg file"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mymod")
  test.write_file(tmpdir + "/mymod/utils.rg", "def ping()\n  return \"pong\"\nend\n")
  script = <<~SCRIPT
    require "mymod"
    puts(mymod.ping())
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "pong")
end

rats "require directory works with rugo build"
  tmpdir = test.tmpdir()
  test.run("cp -r rats/fixtures/dir_entry_lib " + tmpdir + "/dir_entry_lib")
  script = <<~SCRIPT
    require "dir_entry_lib"
    puts(dir_entry_lib.greet("binary"))
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  build_result = test.run("rugo build " + tmpdir + "/main.rg -o " + tmpdir + "/main_bin")
  test.assert_eq(build_result["status"], 0)
  result = test.run(tmpdir + "/main_bin")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello from dir_entry_lib, binary")
end

# --- Negative tests ---

rats "require directory fails when ambiguous (multiple .rg files, no entry point)"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/ambiguous")
  test.write_file(tmpdir + "/ambiguous/foo.rg", "def foo()\n  return 1\nend\n")
  test.write_file(tmpdir + "/ambiguous/bar.rg", "def bar()\n  return 2\nend\n")
  script = <<~SCRIPT
    require "ambiguous"
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "cannot determine entry point")
  test.assert_contains(result["output"], "with")
end

rats "require directory fails when no .rg files"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/empty_dir")
  script = <<~SCRIPT
    require "empty_dir"
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "no .rg files")
end

rats "require prefers file over directory when both exist"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mymod")
  test.write_file(tmpdir + "/mymod.rg", "def source()\n  return \"file\"\nend\n")
  test.write_file(tmpdir + "/mymod/mymod.rg", "def source()\n  return \"dir\"\nend\n")
  script = <<~SCRIPT
    require "mymod"
    puts(mymod.source())
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "file")
end
