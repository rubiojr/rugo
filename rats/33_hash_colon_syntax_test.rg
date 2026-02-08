# RATS: Hash colon syntax (ident: value shorthand)
use "test"
use "conv"

# ============================================================
# A. Basic Colon Syntax
# ============================================================

rats "colon syntax with string value"
  script = <<~SCRIPT
    h = {name: "Alice"}
    puts(h["name"])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Alice")
end

rats "colon syntax with integer value"
  script = <<~SCRIPT
    h = {age: 30}
    puts(h["age"])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "30")
end

rats "colon syntax multiple keys"
  script = <<~SCRIPT
    h = {name: "Alice", age: 30, city: "NYC"}
    puts(h["name"])
    puts(h["age"])
    puts(h["city"])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Alice")
  test.assert_eq(lines[1], "30")
  test.assert_eq(lines[2], "NYC")
end

rats "colon syntax with dot access"
  script = <<~SCRIPT
    h = {name: "Alice", age: 30}
    puts(h.name)
    puts(h.age)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Alice")
  test.assert_eq(lines[1], "30")
end

rats "colon syntax with boolean value"
  script = <<~SCRIPT
    h = {enabled: true, debug: false}
    puts(h.enabled)
    puts(h.debug)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "true")
  test.assert_eq(lines[1], "false")
end

rats "colon syntax with nil value"
  script = <<~SCRIPT
    h = {val: nil}
    puts(h.val == nil)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "true")
end

# ============================================================
# B. Mixed Colon and Arrow Syntax
# ============================================================

rats "mixed colon and arrow keys"
  script = <<~SCRIPT
    h = {name: "Alice", "age" => 30}
    puts(h.name)
    puts(h.age)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Alice")
  test.assert_eq(lines[1], "30")
end

rats "arrow syntax still works unchanged"
  h = {"a" => 1, "b" => 2}
  test.assert_eq(h["a"], 1)
  test.assert_eq(h["b"], 2)
end

# ============================================================
# C. Colon Syntax in Nested Contexts
# ============================================================

rats "nested colon hashes"
  script = <<~SCRIPT
    h = {user: {name: "Alice", email: "alice@test.com"}}
    puts(h.user.name)
    puts(h.user.email)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Alice")
  test.assert_eq(lines[1], "alice@test.com")
end

rats "colon hash in array"
  script = <<~SCRIPT
    items = [{name: "a"}, {name: "b"}]
    puts(items[0].name)
    puts(items[1].name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "a")
  test.assert_eq(lines[1], "b")
end

rats "colon hash returned from function"
  script = <<~SCRIPT
    def make()
      return {name: "Alice", age: 30}
    end
    h = make()
    puts(h.name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Alice")
end

rats "colon syntax with underscore key"
  script = <<~SCRIPT
    h = {first_name: "Alice", last_name: "Smith"}
    puts(h.first_name)
    puts(h.last_name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "Alice")
  test.assert_eq(lines[1], "Smith")
end

# ============================================================
# D. Colon Syntax in Expressions
# ============================================================

rats "colon hash in if condition"
  script = <<~SCRIPT
    config = {enabled: true}
    result = "no"
    if config.enabled
      result = "yes"
    end
    puts(result)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "yes")
end

rats "colon hash with for iteration"
  script = <<~SCRIPT
    h = {x: 10, y: 20}
    total = 0
    for k, v in h
      total += v
    end
    puts(total)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "30")
end

rats "colon hash with dot-set"
  script = <<~SCRIPT
    h = {count: 0}
    h.count = 42
    puts(h.count)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "42")
end

rats "colon hash compiles to native binary"
  script = <<~SCRIPT
    h = {name: "Alice", age: 30}
    puts(h.name)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo build -o " + test.tmpdir() + "/test " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  result = test.run(test.tmpdir() + "/test")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Alice")
end
