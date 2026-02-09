# RATS: Test remote require from git repositories
#
# Starts an in-process git server using web.static + spawn web.listen(0),
# then verifies that require "localhost:PORT/user/repo@version" fetches,
# caches, and compiles remote .rg modules correctly.
use "test"
use "web"
use "conv"
use "str"

# --- File-level setup: bare git repo + in-process server ---

def setup_file()
  test.run("mkdir -p /tmp/rats_remote_require/repos/testuser/rugo-test-mod.git /tmp/rats_remote_require/work")
  test.run("git init --bare /tmp/rats_remote_require/repos/testuser/rugo-test-mod.git")
  test.run("git clone /tmp/rats_remote_require/repos/testuser/rugo-test-mod.git /tmp/rats_remote_require/work")

  mod_src = <<~RG
    def greet(name)
      return "Hello from remote, " + name + "!"
    end

    def double(n)
      return n * 2
    end
  RG
  test.write_file("/tmp/rats_remote_require/work/rugo-test-mod.rg", mod_src)

  test.run("cd /tmp/rats_remote_require/work && git add . && git commit -m initial && git tag v0.1.0")
  test.run("cd /tmp/rats_remote_require/work && git push origin main v0.1.0")
  test.run("cd /tmp/rats_remote_require/repos/testuser/rugo-test-mod.git && git update-server-info")

  web.static("/testuser/rugo-test-mod.git", "/tmp/rats_remote_require/repos/testuser/rugo-test-mod.git")
  spawn web.listen(0)
  p = web.port()
  test.write_file("/tmp/rats_remote_require/port", conv.to_s(p))
end

def teardown_file()
  test.run("rm -rf /tmp/rats_remote_require")
end

# --- Tests ---

rats "remote require loads functions from git repo"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_remote_require/port")["output"]
  consumer = <<~RG
    use "conv"
    require "localhost:PORT/testuser/rugo-test-mod@v0.1.0" as "mod"
    puts(mod.greet("Rugo"))
    puts(conv.to_s(mod.double(21)))
  RG
  consumer = str.replace(consumer, "PORT", port)
  test.write_file(tmpdir + "/consumer.rg", consumer)
  result = test.run("RUGO_MODULE_DIR=" + tmpdir + "/modules rugo run " + tmpdir + "/consumer.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello from remote, Rugo!")
  test.assert_eq(lines[1], "42")
end

rats "remote require uses default namespace from repo name"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_remote_require/port")["output"]
  consumer = <<~RG
    require "localhost:PORT/testuser/rugo-test-mod@v0.1.0"
    puts(rugo_test_mod.greet("world"))
  RG
  consumer = str.replace(consumer, "PORT", port)
  test.write_file(tmpdir + "/consumer.rg", consumer)
  result = test.run("RUGO_MODULE_DIR=" + tmpdir + "/modules rugo run " + tmpdir + "/consumer.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, world!")
end

rats "remote require caches immutable versions"
  tmpdir = test.tmpdir()
  moddir = tmpdir + "/modules"
  port = test.run("cat /tmp/rats_remote_require/port")["output"]
  consumer = <<~RG
    require "localhost:PORT/testuser/rugo-test-mod@v0.1.0" as "mod"
    puts(mod.greet("cache"))
  RG
  consumer = str.replace(consumer, "PORT", port)
  test.write_file(tmpdir + "/consumer.rg", consumer)

  # First run: fetches from server
  result = test.run("RUGO_MODULE_DIR=" + moddir + " rugo run " + tmpdir + "/consumer.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, cache!")

  # Verify cache dir was created
  cache_path = moddir + "/localhost:" + port + "/testuser/rugo-test-mod/v0.1.0"
  result = test.run("test -d " + cache_path)
  test.assert_eq(result["status"], 0)

  # Second run: uses cache (immutable version, compiler skips fetch)
  result = test.run("RUGO_MODULE_DIR=" + moddir + " rugo run " + tmpdir + "/consumer.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, cache!")
end

rats "remote require fails gracefully on bad repo"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_remote_require/port")["output"]
  consumer = <<~RG
    require "localhost:PORT/nobody/nonexistent@v1.0.0" as "bad"
    puts(bad.foo())
  RG
  consumer = str.replace(consumer, "PORT", port)
  test.write_file(tmpdir + "/consumer.rg", consumer)
  result = test.run("RUGO_MODULE_DIR=" + tmpdir + "/modules rugo run " + tmpdir + "/consumer.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "git clone")
end

rats "remote require with build produces working binary"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_remote_require/port")["output"]
  consumer = <<~RG
    use "conv"
    require "localhost:PORT/testuser/rugo-test-mod@v0.1.0" as "mod"
    puts(mod.greet("binary"))
    puts(conv.to_s(mod.double(5)))
  RG
  consumer = str.replace(consumer, "PORT", port)
  test.write_file(tmpdir + "/consumer.rg", consumer)
  test.run("RUGO_MODULE_DIR=" + tmpdir + "/modules rugo build " + tmpdir + "/consumer.rg -o " + tmpdir + "/consumer_bin")
  result = test.run(tmpdir + "/consumer_bin")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello from remote, binary!")
  test.assert_eq(lines[1], "10")
end
