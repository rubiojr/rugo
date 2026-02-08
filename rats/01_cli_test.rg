# RATS: Test basic language features via rugo CLI
use "test"
use "os"

# Test: rugo run with hello world
rats "rugo run prints output"
  result = test.run("rugo run examples/hello.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end

# Test: rugo build produces a binary
rats "rugo build creates binary"
  result = test.run("rugo build -o " + test.tmpdir() + "/hello examples/hello.rg")
  test.assert_eq(result["status"], 0)
  result = test.run(test.tmpdir() + "/hello")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end

# Test: rugo emit outputs Go source
rats "rugo emit outputs valid Go"
  result = test.run("rugo emit examples/hello.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package main")
  test.assert_contains(result["output"], "func main()")
end

# Test: rugo version flag
rats "rugo --version works"
  result = test.run("rugo --version")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugo version")
end

# Test: rugo help flag
rats "rugo --help works"
  result = test.run("rugo --help")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "COMMANDS")
  test.assert_contains(result["output"], "run")
  test.assert_contains(result["output"], "build")
  test.assert_contains(result["output"], "emit")
  test.assert_contains(result["output"], "test")
end

# Test: rugo with no args shows help
rats "rugo with no args shows help"
  result = test.run("rugo")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "COMMANDS")
end

# Test: rugo run with nonexistent file
rats "rugo run with missing file fails"
  result = test.run("rugo run /tmp/nonexistent_file_12345.rg")
  test.assert_neq(result["status"], 0)
end

# Test: shorthand rugo script.rg (without run subcommand)
rats "rugo script.rg shorthand works"
  result = test.run("rugo examples/hello.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Hello")
end
