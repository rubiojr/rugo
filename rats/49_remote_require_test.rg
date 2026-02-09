# RATS: Test remote require from git repositories
#
# Spins up a local git server using web.static, then verifies that
# require "localhost:PORT/user/repo@version" fetches, caches, and
# compiles remote .rg modules correctly.
use "test"
use "str"

# Helper: create a bare git repo, start a Rugo HTTP server, run a consumer
# script, and return the consumer result. Server lifecycle is managed within
# a single shell invocation to avoid orphan processes.
def run_with_git_server(port, consumer_src)
  tmpdir = test.tmpdir()
  bare = tmpdir + "/repos/testuser/rugo-test-mod.git"
  work = tmpdir + "/work"

  test.run("mkdir -p " + bare)
  test.run("git init --bare " + bare)
  test.run("git clone " + bare + " " + work)

  mod_src = <<~RG
    def greet(name)
      return "Hello from remote, " + name + "!"
    end

    def double(n)
      return n * 2
    end
  RG
  test.write_file(work + "/rugo-test-mod.rg", mod_src)

  test.run("cd " + work + " && git add . && git commit -m initial && git tag v0.1.0")
  test.run("cd " + work + " && git push origin main v0.1.0")
  test.run("cd " + bare + " && git update-server-info")

  server_src = <<~RG
    use "web"
    web.static("/testuser/rugo-test-mod.git", "BARE")
    web.listen(PORT)
  RG
  server_src = str.replace(server_src, "BARE", bare)
  server_src = str.replace(server_src, "PORT", port)
  test.write_file(tmpdir + "/server.rg", server_src)
  test.write_file(tmpdir + "/consumer.rg", consumer_src)

  # Build server binary, then run it directly so kill works on the right PID
  runner = tmpdir + "/run.sh"
  script = "#!/bin/sh\nrugo build " + tmpdir + "/server.rg -o " + tmpdir + "/server_bin\n"
  script = script + tmpdir + "/server_bin &\nSERVER_PID=$!\nsleep 2\n"
  script = script + "RUGO_MODULE_DIR=" + tmpdir + "/modules rugo run " + tmpdir + "/consumer.rg 2>&1\n"
  script = script + "EXIT=$?\nkill $SERVER_PID 2>/dev/null\nwait $SERVER_PID 2>/dev/null\nexit $EXIT\n"
  test.write_file(runner, script)

  return test.run("sh " + runner)
end

# --- Tests ---

rats "remote require loads functions from git repo"
  consumer = <<~RG
    use "conv"
    require "localhost:19360/testuser/rugo-test-mod@v0.1.0" as "mod"
    puts(mod.greet("Rugo"))
    puts(conv.to_s(mod.double(21)))
  RG
  result = run_with_git_server("19360", consumer)
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello from remote, Rugo!")
  test.assert_eq(lines[1], "42")
end

rats "remote require uses default namespace from repo name"
  consumer = <<~RG
    require "localhost:19361/testuser/rugo-test-mod@v0.1.0"
    puts(rugo_test_mod.greet("world"))
  RG
  result = run_with_git_server("19361", consumer)
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, world!")
end

rats "remote require caches immutable versions"
  tmpdir = test.tmpdir()
  bare = tmpdir + "/repos/testuser/rugo-test-mod.git"
  work = tmpdir + "/work"
  port = "19362"

  test.run("mkdir -p " + bare)
  test.run("git init --bare " + bare)
  test.run("git clone " + bare + " " + work)
  mod_src = "def greet(name)\n  return \"Hello from remote, \" + name + \"!\"\nend\n"
  test.write_file(work + "/rugo-test-mod.rg", mod_src)
  test.run("cd " + work + " && git add . && git commit -m initial && git tag v0.1.0")
  test.run("cd " + work + " && git push origin main v0.1.0")
  test.run("cd " + bare + " && git update-server-info")

  server_src = "use \"web\"\nweb.static(\"/testuser/rugo-test-mod.git\", \"" + bare + "\")\nweb.listen(" + port + ")\n"
  test.write_file(tmpdir + "/server.rg", server_src)

  consumer_src = "require \"localhost:" + port + "/testuser/rugo-test-mod@v0.1.0\" as \"mod\"\nputs(mod.greet(\"cache\"))\n"
  test.write_file(tmpdir + "/consumer.rg", consumer_src)

  moddir = tmpdir + "/modules"

  # First run: with server (build server binary to avoid orphan processes)
  runner = "#!/bin/sh\nrugo build " + tmpdir + "/server.rg -o " + tmpdir + "/server_bin\n"
  runner = runner + tmpdir + "/server_bin &\nSERVER_PID=$!\nsleep 2\n"
  runner = runner + "RUGO_MODULE_DIR=" + moddir + " rugo run " + tmpdir + "/consumer.rg 2>&1\n"
  runner = runner + "EXIT=$?\nkill $SERVER_PID 2>/dev/null\nwait $SERVER_PID 2>/dev/null\nexit $EXIT\n"
  test.write_file(tmpdir + "/run.sh", runner)
  result = test.run("sh " + tmpdir + "/run.sh")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, cache!")

  # Second run: server is dead, should use cache (immutable version)
  result = test.run("RUGO_MODULE_DIR=" + moddir + " rugo run " + tmpdir + "/consumer.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello from remote, cache!")
end

rats "remote require fails gracefully on bad repo"
  tmpdir = test.tmpdir()

  consumer = <<~RG
    require "localhost:19363/nobody/nonexistent@v1.0.0" as "bad"
    puts(bad.foo())
  RG
  test.write_file(tmpdir + "/consumer.rg", consumer)

  result = test.run("RUGO_MODULE_DIR=" + tmpdir + "/modules rugo run " + tmpdir + "/consumer.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "git clone")
end

rats "remote require with build produces working binary"
  consumer = <<~RG
    use "conv"
    require "localhost:19364/testuser/rugo-test-mod@v0.1.0" as "mod"
    puts(mod.greet("binary"))
    puts(conv.to_s(mod.double(5)))
  RG
  result = run_with_git_server("19364", consumer)
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Hello from remote, binary!")
  test.assert_eq(lines[1], "10")
end
