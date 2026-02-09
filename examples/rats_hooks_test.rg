# Example: RATS setup_file/teardown_file hooks
#
# setup_file() runs once before all tests in the file.
# teardown_file() runs once after all tests (even on failure).
# setup() runs before each individual test.
# teardown() runs after each individual test.
use "test"
use "os"

def setup_file()
  os.exec("mkdir -p /tmp/rats_hooks_example")
  test.write_file("/tmp/rats_hooks_example/greeting.txt", "Hello from setup_file!")
end

def teardown_file()
  os.exec("rm -rf /tmp/rats_hooks_example")
end

def setup()
  test.write_file(test.tmpdir() + "/per_test.txt", "fresh")
end

rats "setup_file creates shared resources"
  result = test.run("cat /tmp/rats_hooks_example/greeting.txt")
  test.assert_eq(result["output"], "Hello from setup_file!")
end

rats "setup resets per-test state"
  result = test.run("cat " + test.tmpdir() + "/per_test.txt")
  test.assert_eq(result["output"], "fresh")
end
