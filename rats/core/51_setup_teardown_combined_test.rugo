# RATS: Test setup_file combined with per-test setup/teardown
use "test"
use "os"
use "conv"

def setup_file()
  os.exec("mkdir -p /tmp/rats_combined_test")
  test.write_file("/tmp/rats_combined_test/counter.txt", "0")
end

def setup()
  result = test.run("cat /tmp/rats_combined_test/counter.txt")
  count = conv.to_i(result["output"]) + 1
  test.write_file("/tmp/rats_combined_test/counter.txt", conv.to_s(count))
end

def teardown()
  result = test.run("cat /tmp/rats_combined_test/counter.txt")
  test.assert_neq(result["output"], "")
end

def teardown_file()
  os.exec("rm -rf /tmp/rats_combined_test")
end

rats "first test sees counter at 1"
  result = test.run("cat /tmp/rats_combined_test/counter.txt")
  test.assert_eq(result["output"], "1")
end

rats "second test sees counter at 2"
  result = test.run("cat /tmp/rats_combined_test/counter.txt")
  test.assert_eq(result["output"], "2")
end

rats "third test sees counter at 3"
  result = test.run("cat /tmp/rats_combined_test/counter.txt")
  test.assert_eq(result["output"], "3")
end
