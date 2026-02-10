# RATS: Test rugo doc command
use "test"

# Test: rugo doc with no args lists all modules
rats "rugo doc lists all modules and packages"
  result = test.run("rugo doc")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Modules (use):")
  test.assert_contains(result["output"], "Bridge packages (import):")
  test.assert_contains(result["output"], "http")
  test.assert_contains(result["output"], "strings")
end

# Test: rugo doc --all also lists everything
rats "rugo doc --all lists everything"
  result = test.run("rugo doc --all")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Modules (use):")
  test.assert_contains(result["output"], "Bridge packages (import):")
end

# Test: rugo doc for a stdlib module
rats "rugo doc http shows module docs"
  result = test.run("rugo doc http")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "module http")
  test.assert_contains(result["output"], "http.get")
  test.assert_contains(result["output"], "http.post")
end

# Test: rugo doc for a bridge package
rats "rugo doc time shows bridge docs"
  result = test.run("rugo doc time")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package time")
  test.assert_contains(result["output"], "time.now_unix")
  test.assert_contains(result["output"], "time.sleep_ms")
end

# Test: rugo doc strings shows all bridge functions
rats "rugo doc strings shows bridge functions"
  result = test.run("rugo doc strings")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package strings")
  test.assert_contains(result["output"], "strings.contains")
  test.assert_contains(result["output"], "strings.split")
  test.assert_contains(result["output"], "strings.trim_space")
end

# Test: rugo doc for a .rg file
rats "rugo doc on .rg file extracts docs"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/lib.rg", "# A helper library.\n\n# Adds two numbers.\ndef add(a, b)\n  return a + b\nend\n")
  result = test.run("rugo doc " + tmpdir + "/lib.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "A helper library.")
  test.assert_contains(result["output"], "def add(a, b)")
  test.assert_contains(result["output"], "Adds two numbers.")
end

# Test: rugo doc for a specific symbol
rats "rugo doc file symbol looks up function"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/lib.rg", "# Multiplies x by y.\ndef mul(x, y)\n  return x * y\nend\n\ndef other()\nend\n")
  result = test.run("rugo doc " + tmpdir + "/lib.rg mul")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def mul(x, y)")
  test.assert_contains(result["output"], "Multiplies x by y.")
end

# Test: rugo doc for struct
rats "rugo doc file symbol looks up struct"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/lib.rg", "# A 2D point.\nstruct Point\n  x\n  y\nend\n")
  result = test.run("rugo doc " + tmpdir + "/lib.rg Point")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "struct Point")
  test.assert_contains(result["output"], "A 2D point.")
end

# Test: rugo doc with missing symbol
rats "rugo doc file with missing symbol fails"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/lib.rg", "def foo()\nend\n")
  result = test.run("rugo doc " + tmpdir + "/lib.rg missing")
  test.assert_neq(result["status"], 0)
end

# Test: rugo doc with unknown module
rats "rugo doc unknown module fails"
  result = test.run("rugo doc nonexistent_module_xyz")
  test.assert_neq(result["status"], 0)
end

# Test: rugo doc blank line breaks doc attachment
rats "rugo doc blank line breaks attachment"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/lib.rg", "# Orphaned comment.\n\ndef foo()\nend\n")
  result = test.run("rugo doc " + tmpdir + "/lib.rg foo")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def foo")
end

# Test: rugo doc filepath bridge package
rats "rugo doc filepath shows bridge docs"
  result = test.run("rugo doc filepath")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package filepath")
  test.assert_contains(result["output"], "filepath.join")
  test.assert_contains(result["output"], "filepath.base")
end

# Test: rugo doc math bridge package
rats "rugo doc math shows bridge docs"
  result = test.run("rugo doc math")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package math")
  test.assert_contains(result["output"], "math.sqrt")
  test.assert_contains(result["output"], "math.pow")
end

# Test: rugo doc with NO_COLOR skips bat
rats "rugo doc respects NO_COLOR"
  result = test.run("NO_COLOR=1 rugo doc time")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "package time")
  test.assert_contains(result["output"], "time.sleep_ms")
end

# Test: rugo doc with invalid remote path fails gracefully
rats "rugo doc invalid remote fails"
  result = test.run("rugo doc invalid.host/no/repo")
  test.assert_neq(result["status"], 0)
end

# Test: file-level doc captured when comment block runs directly into code
rats "rugo doc file-level doc before code without blank line"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/mod.rg", "# Module docs.\n# Second line.\nrequire \"./lib\"\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/mod.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Module docs.")
  test.assert_contains(result["output"], "Second line.")
end

# Test: rugo doc shows structs with fields
rats "rugo doc shows struct fields"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/t.rg", "# A Config.\nstruct Config\n  host\n  port\n  debug\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/t.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "struct Config")
  test.assert_contains(result["output"], "host")
  test.assert_contains(result["output"], "A Config.")
end

# Test: rugo doc method on struct (def Struct.method)
rats "rugo doc shows struct methods"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/t.rg", "struct Dog\n  name\nend\n\n# Makes the dog bark.\ndef Dog.bark()\n  return self.name\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/t.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def Dog.bark")
  test.assert_contains(result["output"], "Makes the dog bark.")
end

# Test: rugo doc multiline doc comment preserved
rats "rugo doc multiline doc"
  tmpdir = test.tmpdir()
  test.write_file(tmpdir + "/t.rg", "# Line one.\n# Line two.\n# Line three.\ndef multi()\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/t.rg multi")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Line one.")
  test.assert_contains(result["output"], "Line two.")
  test.assert_contains(result["output"], "Line three.")
end

# Test: rugo doc all bridge packages have docs
rats "rugo doc shows all bridge packages"
  result = test.run("NO_COLOR=1 rugo doc --all")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "math")
  test.assert_contains(result["output"], "rand")
  test.assert_contains(result["output"], "filepath")
  test.assert_contains(result["output"], "sort")
  test.assert_contains(result["output"], "strconv")
  test.assert_contains(result["output"], "time")
end

# Test: rugo doc conv module functions
rats "rugo doc conv shows module functions"
  result = test.run("NO_COLOR=1 rugo doc conv")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "module conv")
  test.assert_contains(result["output"], "conv.to_i")
  test.assert_contains(result["output"], "conv.to_f")
  test.assert_contains(result["output"], "conv.to_s")
end

# Test: rugo doc strconv bridge has typed signatures
rats "rugo doc strconv shows typed signatures"
  result = test.run("NO_COLOR=1 rugo doc strconv")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "strconv.atoi(string)")
  test.assert_contains(result["output"], "-> int")
  test.assert_contains(result["output"], "strconv.itoa(int)")
  test.assert_contains(result["output"], "-> string")
end

# Test: rugo doc sort bridge
rats "rugo doc sort shows functions"
  result = test.run("NO_COLOR=1 rugo doc sort")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "sort.strings")
  test.assert_contains(result["output"], "sort.ints")
  test.assert_contains(result["output"], "sort.is_sorted")
end

# Test: rugo doc rand bridge
rats "rugo doc rand shows functions"
  result = test.run("NO_COLOR=1 rugo doc rand")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rand.int_n")
  test.assert_contains(result["output"], "rand.float64")
end

# Test: rugo doc example file works
rats "rugo doc examples/documented.rg works"
  result = test.run("NO_COLOR=1 rugo doc examples/documented.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "Math utility library")
  test.assert_contains(result["output"], "def factorial(n)")
  test.assert_contains(result["output"], "def greet(name)")
end

# Test: rugo doc on local directory with entry point
rats "rugo doc local directory shows docs"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mymod")
  test.write_file(tmpdir + "/mymod/mymod.rg", "# My module.\n\n# Does stuff.\ndef do_stuff()\n  return 1\nend\n")
  test.write_file(tmpdir + "/mymod/helpers.rg", "# A helper.\ndef helper()\n  return 2\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/mymod")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "My module.")
  test.assert_contains(result["output"], "def do_stuff")
  test.assert_contains(result["output"], "def helper")
end

# Test: rugo doc on local directory with symbol lookup
rats "rugo doc local directory symbol lookup"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mymod")
  test.write_file(tmpdir + "/mymod/mymod.rg", "# Adds numbers.\ndef add(a, b)\n  return a + b\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/mymod add")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "def add(a, b)")
  test.assert_contains(result["output"], "Adds numbers.")
end

# Test: rugo doc on local directory with missing symbol fails
rats "rugo doc local directory missing symbol fails"
  tmpdir = test.tmpdir()
  test.run("mkdir -p " + tmpdir + "/mymod")
  test.write_file(tmpdir + "/mymod/mymod.rg", "def foo()\nend\n")
  result = test.run("NO_COLOR=1 rugo doc " + tmpdir + "/mymod nope")
  test.assert_neq(result["status"], 0)
end
