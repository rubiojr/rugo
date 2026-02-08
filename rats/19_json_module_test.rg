# RATS: Test json module (parse, encode)
use "test"
use "conv"

rats "json.parse parses object"
  script = <<~SCRIPT
    use "json"
    use "conv"
    data = json.parse("{\"name\": \"rugo\", \"version\": 1}")
    puts(data["name"])
    puts(conv.to_s(data["version"]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "rugo")
  test.assert_eq(lines[1], "1")
end

rats "json.parse parses array"
  script = <<~SCRIPT
    use "json"
    use "conv"
    arr = json.parse("[1, 2, 3]")
    puts(conv.to_s(len(arr)))
    puts(conv.to_s(arr[0]))
    puts(conv.to_s(arr[2]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "3")
  test.assert_eq(lines[1], "1")
  test.assert_eq(lines[2], "3")
end

rats "json.parse converts whole numbers to int"
  script = <<~SCRIPT
    use "json"
    use "conv"
    data = json.parse("{\"id\": 12345}")
    puts(conv.to_s(data["id"]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "12345")
end

rats "json.parse preserves floats"
  script = <<~SCRIPT
    use "json"
    use "conv"
    data = json.parse("{\"pi\": 3.14}")
    puts(conv.to_s(data["pi"]))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "3.14")
end

rats "json.parse handles nested objects"
  script = <<~SCRIPT
    use "json"
    data = json.parse("{\"user\": {\"name\": \"rugo\"}}")
    puts(data["user"]["name"])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "rugo")
end

rats "json.parse handles booleans and null"
  script = <<~SCRIPT
    use "json"
    use "conv"
    data = json.parse("{\"ok\": true, \"err\": null}")
    puts(conv.to_s(data["ok"]))
    puts(data["err"] == nil)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "true")
  test.assert_eq(lines[1], "true")
end

rats "json.encode converts hash to JSON"
  script = <<~SCRIPT
    use "json"
    h = {"a" => 1}
    puts(json.encode(h))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"a\"")
  test.assert_contains(result["output"], "1")
end

rats "json.encode converts array to JSON"
  script = <<~SCRIPT
    use "json"
    arr = [1, "two", true]
    puts(json.encode(arr))
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "[1,\"two\",true]")
end

rats "json.parse roundtrip"
  script = <<~SCRIPT
    use "json"
    original = "[1,2,3]"
    parsed = json.parse(original)
    result = json.encode(parsed)
    puts(result)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "[1,2,3]")
end

rats "json.parse panics on invalid JSON"
  script = <<~SCRIPT
    use "json"
    json.parse("not json")
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_neq(result["status"], 0)
  test.assert_contains(result["output"], "json.parse")
end
