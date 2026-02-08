# RATS: Module import/require edge cases
import "test"
import "str"

# Bug 186d619: require alias matching stdlib name without import should use user module
rats "require alias with stdlib name calls user module"
  test.run("mkdir -p " + test.tmpdir() + "/t1")
  test.run("printf 'def upper(s)\nreturn \"CUSTOM: \" + s\nend\n' > " + test.tmpdir() + "/t1/helpers.rg")
  test.run("printf 'require \"helpers\" as \"str\"\nputs(str.upper(\"hello\"))\n' > " + test.tmpdir() + "/t1/main.rg")
  result = test.run("rugo run " + test.tmpdir() + "/t1/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "CUSTOM: hello")
end

# Bug 65f41d8: require alias conflicting with imported module should error
rats "require alias conflicts with imported module"
  test.run("mkdir -p " + test.tmpdir() + "/t2")
  test.run("printf 'def upper(s)\nreturn \"CUSTOM\"\nend\n' > " + test.tmpdir() + "/t2/helpers.rg")
  test.run("printf 'import \"str\"\nrequire \"helpers\" as \"str\"\n' > " + test.tmpdir() + "/t2/main.rg")
  result = test.run("rugo run " + test.tmpdir() + "/t2/main.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "str")
end

# Bug 6013a24: imports inside required files should propagate
rats "required file imports propagate"
  test.run("mkdir -p " + test.tmpdir() + "/t3")
  test.run("printf 'import \"conv\"\ndef double_str(n)\nreturn conv.to_s(n * 2)\nend\n' > " + test.tmpdir() + "/t3/helpers.rg")
  test.run("printf 'require \"helpers\"\nputs(helpers.double_str(21))\n' > " + test.tmpdir() + "/t3/main.rg")
  result = test.run("rugo run " + test.tmpdir() + "/t3/main.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

# Bug 51ab700: import inside function body should error
rats "import inside function body errors"
  test.run("printf 'def foo()\nimport \"conv\"\nend\n' > " + test.tmpdir() + "/nested_import.rg")
  result = test.run("rugo run " + test.tmpdir() + "/nested_import.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "top level")
end

# Bug 51ab700: require inside function body should error
rats "require inside function body errors"
  test.run("mkdir -p " + test.tmpdir() + "/t5")
  test.run("printf 'def foo()\nreturn 1\nend\n' > " + test.tmpdir() + "/t5/helpers.rg")
  test.run("printf 'def bar()\nrequire \"helpers\"\nend\n' > " + test.tmpdir() + "/t5/main.rg")
  result = test.run("rugo run " + test.tmpdir() + "/t5/main.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "top level")
end

# Bug 59e1dc8: two requires same alias with conflicting function names
rats "duplicate namespace function errors"
  test.run("mkdir -p " + test.tmpdir() + "/t6")
  test.run("printf 'def foo()\nreturn \"a\"\nend\n' > " + test.tmpdir() + "/t6/a.rg")
  test.run("printf 'def foo()\nreturn \"b\"\nend\n' > " + test.tmpdir() + "/t6/b.rg")
  test.run("printf 'require \"a\" as \"ns\"\nrequire \"b\" as \"ns\"\nputs(ns.foo())\n' > " + test.tmpdir() + "/t6/main.rg")
  result = test.run("rugo run " + test.tmpdir() + "/t6/main.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "foo")
end

# Bug 6ee382f: duplicate import is silently deduplicated
rats "duplicate import is deduplicated"
  test.run("printf 'import \"conv\"\nimport \"conv\"\nputs(conv.to_s(42))\n' > " + test.tmpdir() + "/dup_import.rg")
  result = test.run("rugo run " + test.tmpdir() + "/dup_import.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end
