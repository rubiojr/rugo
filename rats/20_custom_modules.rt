# RATS: Test custom modules (external module support)
import "test"

rats "custom rugo binary builds"
  result = test.run("cd rats/fixtures/custom_module/custom-rugo && go build -o " + test.tmpdir() + "/custom-rugo .")
  test.assert_eq(result["status"], 0)
end

rats "custom module hello.greet works"
  test.run("cd rats/fixtures/custom_module/custom-rugo && go build -o " + test.tmpdir() + "/custom-rugo .")
  result = test.run(test.tmpdir() + "/custom-rugo rats/fixtures/custom_module/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "hello, rugo")
  test.assert_eq(lines[1], "hello, world!")
end

rats "custom rugo version shows custom string"
  test.run("cd rats/fixtures/custom_module/custom-rugo && go build -o " + test.tmpdir() + "/custom-rugo .")
  result = test.run(test.tmpdir() + "/custom-rugo --version")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "v0.0.0-test")
end

rats "custom rugo still runs standard modules"
  test.run("cd rats/fixtures/custom_module/custom-rugo && go build -o " + test.tmpdir() + "/custom-rugo .")
  test.run("printf 'import \"str\"\nputs(str.upper(\"hello\"))\n' > " + test.tmpdir() + "/stdlib.rg")
  result = test.run(test.tmpdir() + "/custom-rugo " + test.tmpdir() + "/stdlib.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "HELLO")
end
