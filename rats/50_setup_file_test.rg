# RATS: Test setup_file and teardown_file hooks
use "test"
use "os"

def setup_file()
  os.exec("mkdir -p /tmp/rats_setup_file_test")
  test.write_file("/tmp/rats_setup_file_test/marker.txt", "setup_file_called")
end

def teardown_file()
  os.exec("rm -rf /tmp/rats_setup_file_test")
end

rats "setup_file runs before tests"
  result = test.run("cat /tmp/rats_setup_file_test/marker.txt")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "setup_file_called")
end

rats "setup_file state persists across tests"
  result = test.run("cat /tmp/rats_setup_file_test/marker.txt")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "setup_file_called")
end

