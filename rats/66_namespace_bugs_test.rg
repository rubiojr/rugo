# RATS: Namespace resolution bugs
# Bug 1: barrel file (dir entry point with only requires) produces no namespace
# Bug 2: variable assignment should shadow require namespace in codegen
use "test"

rats "barrel file require exposes inner namespaces"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mylib")
  test.write_file(tmpdir + "/mylib/mylib.rg", "require \"inner\"\n")
  test.write_file(tmpdir + "/mylib/inner.rg", "def greet(name)\n  return \"hi \" + name\nend\n")
  script = <<~SCRIPT
    require "mylib"
    puts(inner.greet("world"))
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hi world")
end

rats "variable shadows require namespace for dot calls"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/helper.rg", "def make()\n  return {foo: \"bar\"}\nend\n")
  script = <<~SCRIPT
    require "helper"
    helper = helper.make()
    puts(helper.foo)
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "bar")
end

rats "variable shadows namespace - no internal Go symbols leaked"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/helper.rg", "def make()\n  return {foo: \"bar\"}\nend\n")
  script = "require \"helper\"\nhelper = helper.make()\nputs(helper.foo)\n"
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  # Must succeed — no rugons_ symbol leak
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "bar")
end

rats "variable does not shadow namespace before assignment"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/helper.rg", "def greet()\n  return \"hello\"\nend\n")
  script = <<~SCRIPT
    require "helper"
    puts(helper.greet())
  SCRIPT
  test.write_file(tmpdir + "/main.rg", script)
  result = test.run("rugo run " + tmpdir + "/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello")
end
