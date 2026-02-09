# RATS: Integration test for rugo doc with remote modules
#
# Sets up a local git server (web.static + spawn web.listen(0)) with a
# multi-file Rugo module and tests that `rugo doc` aggregates docs from
# all .rg files in the remote module.
use "test"
use "web"
use "conv"
use "str"

# --- File-level setup: bare git repo + in-process server ---

def setup_file()
  r = test.run("rm -rf /tmp/rats_doc_remote && mkdir -p /tmp/rats_doc_remote/repos/testuser/docmod.git /tmp/rats_doc_remote/work")
  if r["status"] != 0
    puts "DEBUG setup mkdir: " + r["output"]
  end
  r = test.run("git init --bare /tmp/rats_doc_remote/repos/testuser/docmod.git")
  if r["status"] != 0
    puts "DEBUG setup git init: " + r["output"]
  end
  r = test.run("git clone /tmp/rats_doc_remote/repos/testuser/docmod.git /tmp/rats_doc_remote/work")
  if r["status"] != 0
    puts "DEBUG setup git clone: " + r["output"]
  end

  # main.rg — entry point with file-level doc, no blank line before require
  main_src = <<~RG
    # docmod — A documented test module.
    # This module demonstrates doc extraction.
    require "./helpers"
    require "./types"
  RG
  test.write_file("/tmp/rats_doc_remote/work/docmod.rg", main_src)

  # helpers.rg — functions with doc comments
  helpers_src = <<~RG
    # Adds two numbers together.
    def add(a, b)
      return a + b
    end

    # Greets a user by name.
    def greet(name)
      return "Hello, " + name + "!"
    end

    def undocumented()
    end
  RG
  test.write_file("/tmp/rats_doc_remote/work/helpers.rg", helpers_src)

  # types.rg — struct with doc comments
  types_src = <<~RG
    # A 2D point with x and y coordinates.
    struct Point
      x
      y
    end
  RG
  test.write_file("/tmp/rats_doc_remote/work/types.rg", types_src)

  r = test.run("cd /tmp/rats_doc_remote/work && git config user.email test@test.com && git config user.name test && git add . && git commit -m initial && git tag v1.0.0")
  if r["status"] != 0
    puts "DEBUG setup git commit: " + r["output"]
  end
  r = test.run("cd /tmp/rats_doc_remote/work && git push origin HEAD v1.0.0")
  if r["status"] != 0
    puts "DEBUG setup git push: " + r["output"]
  end
  r = test.run("cd /tmp/rats_doc_remote/repos/testuser/docmod.git && git update-server-info")
  if r["status"] != 0
    puts "DEBUG setup update-server-info: " + r["output"]
  end

  web.static("/testuser/docmod.git", "/tmp/rats_doc_remote/repos/testuser/docmod.git")
  spawn web.listen(0)
  p = web.port()
  test.write_file("/tmp/rats_doc_remote/port", conv.to_s(p))
end

def teardown_file()
  test.run("rm -rf /tmp/rats_doc_remote")
end

# --- Tests ---

rats "rugo doc remote shows file-level doc"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0")
  if result["status"] != 0
    puts "DEBUG output: " + result["output"]
  end
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "docmod")
  test.assert_contains(result["output"], "documented test module")
end

rats "rugo doc remote aggregates functions from all files"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def add(a, b)")
  test.assert_contains(result["output"], "Adds two numbers")
  test.assert_contains(result["output"], "def greet(name)")
  test.assert_contains(result["output"], "Greets a user")
end

rats "rugo doc remote aggregates structs from all files"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "struct Point")
  test.assert_contains(result["output"], "2D point")
end

rats "rugo doc remote symbol lookup across files"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0 add")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def add(a, b)")
  test.assert_contains(result["output"], "Adds two numbers")
end

rats "rugo doc remote symbol lookup for struct"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0 Point")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "struct Point")
end

rats "rugo doc remote missing symbol fails"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0 nonexistent")
  test.assert_neq(result["status"], 0)
end

rats "rugo doc remote undocumented functions show without doc"
  tmpdir = test.tmpdir()
  port = test.run("cat /tmp/rats_doc_remote/port")["output"]
  result = test.run("NO_COLOR=1 RUGO_MODULE_DIR=" + tmpdir + "/modules rugo doc localhost:" + port + "/testuser/docmod@v1.0.0")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def undocumented")
end
