# RATS: Test web module (routing, middleware, responses)
use "test"

# --- Basic routing ---

rats "web.get serves text response"
  result = test.run("rugo run rats/fixtures/web_basic.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, World!")
end

rats "web.get with URL parameters"
  result = test.run("rugo run rats/fixtures/web_params.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "user:42")
  test.assert_eq(lines[1], "user:hello")
end

rats "web.get with multiple URL parameters"
  result = test.run("rugo run rats/fixtures/web_multi_params.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "5:99")
end

rats "web.get root path"
  result = test.run("rugo run rats/fixtures/web_root.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "root")
end

# --- Response helpers ---

rats "web.json returns JSON object"
  result = test.run("rugo run rats/fixtures/web_json.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"name\":\"rugo\"")
  test.assert_contains(result["output"], "\"version\":1")
end

rats "web.json returns JSON array"
  result = test.run("rugo run rats/fixtures/web_json_array.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "[1,2,3]")
end

rats "web.html returns HTML content"
  result = test.run("rugo run rats/fixtures/web_html.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "<h1>Hello</h1>")
end

rats "web.status returns empty body"
  result = test.run("rugo run rats/fixtures/web_status.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "body:")
end

# --- HTTP methods ---

rats "web.post handles POST requests"
  result = test.run("rugo run rats/fixtures/web_post.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"created\":true")
  test.assert_contains(result["output"], "\"body\":\"{\\\"name\\\":\\\"test\\\"}\"")
end

rats "web supports multiple HTTP methods"
  result = test.run("rugo run rats/fixtures/web_methods.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "GET")
  test.assert_eq(lines[1], "POST")
  test.assert_eq(lines[2], "PUT")
  test.assert_eq(lines[3], "DELETE")
  test.assert_eq(lines[4], "PATCH")
end

# --- Request object ---

rats "request dot access works"
  result = test.run("rugo run rats/fixtures/web_dot_access.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "GET /info")
end

rats "request query parameters"
  result = test.run("rugo run rats/fixtures/web_query.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "rugo:10")
end

rats "request body parsing with json"
  result = test.run("rugo run rats/fixtures/web_body_parse.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"received\":\"hello\"")
end

# --- Middleware ---

rats "global middleware can block requests"
  result = test.run("rugo run rats/fixtures/web_middleware_block.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "blocked")
end

rats "global middleware can pass through"
  result = test.run("rugo run rats/fixtures/web_middleware_pass.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "reached handler")
end

rats "route-level middleware"
  result = test.run("rugo run rats/fixtures/web_route_middleware.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "public")
  test.assert_eq(lines[1], "unauthorized")
end

# --- Route groups ---

rats "route groups with prefix"
  result = test.run("rugo run rats/fixtures/web_group.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "home")
  test.assert_eq(lines[1], "users")
  test.assert_eq(lines[2], "posts")
end

rats "route groups with middleware"
  result = test.run("rugo run rats/fixtures/web_group_middleware.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "public")
  test.assert_eq(lines[1], "forbidden")
end

# --- Inline script tests for advanced features ---

rats "web.json with status code 201"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    use "json"
    web.post("/items", "create_item")
    def create_item(req)
      return web.json({"ok" => true}, 201)
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.post("http://localhost:#{_port}/items", "{}").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"ok\":true")
end

rats "web.text with status code 404"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/missing", "missing_handler")
    def missing_handler(req)
      return web.text("not found", 404)
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/missing").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "not found")
end

rats "nested hash in JSON response"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/nested", "nested_handler")
    def nested_handler(req)
      return web.json({"user" => {"name" => "Alice", "age" => 30}})
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/nested").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"name\":\"Alice\"")
  test.assert_contains(result["output"], "\"age\":30")
end

rats "handler receives correct method for POST"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.post("/check", "check_method")
    def check_method(req)
      return web.text(req.method)
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.post("http://localhost:#{_port}/check", "").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "POST")
end

rats "middleware chain runs in order"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.middleware("mw_first")
    web.middleware("mw_second")
    web.get("/chain", "chain_handler")
    def mw_first(req)
      return nil
    end
    def mw_second(req)
      return web.text("stopped by mw_second")
    end
    def chain_handler(req)
      return web.text("should not reach")
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/chain").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "stopped by mw_second")
end

rats "built-in logger middleware runs without error"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.middleware("logger")
    web.get("/logged", "logged_handler")
    def logged_handler(req)
      return web.text("ok")
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/logged").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "ok")
  test.assert_contains(result["output"], "GET /logged")
end

rats "custom headers on response"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/custom", "custom_handler")
    def custom_handler(req)
      resp = web.text("with headers")
      resp["headers"] = {"X-Custom" => "hello"}
      return resp
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/custom").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "with headers")
end

rats "web.group and end_group reset prefix"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.group("/api")
      web.get("/items", "items_handler")
    web.end_group()
    web.get("/outside", "outside_handler")
    def items_handler(req)
      return web.text("items")
    end
    def outside_handler(req)
      return web.text("outside")
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/api/items").body)
    puts(http.get("http://localhost:#{_port}/outside").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "items")
  test.assert_eq(lines[1], "outside")
end

rats "handler with string interpolation"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/greet/:name", "greet_handler")
    def greet_handler(req)
      name = req.params["name"]
      return web.text("Hello, #{name}!")
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/greet/Rugo").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Hello, Rugo!")
end

rats "web module compiles to native binary"
  script = <<~SCRIPT
    use "web"
    web.get("/", "home")
    def home(req)
      return web.text("ok")
    end
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo build " + test.tmpdir() + "/test.rg -o " + test.tmpdir() + "/webbin")
  test.assert_eq(result["status"], 0)
  test.run("rm -f " + test.tmpdir() + "/webbin")
end

rats "web emit includes dispatch map"
  script = <<~SCRIPT
    use "web"
    web.get("/", "home")
    def home(req)
      return web.text("ok")
    end
    web.listen(9999)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo emit " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "rugo_web_dispatch")
  test.assert_contains(result["output"], "rugofn_home")
end

rats "multiple routes to different handlers"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/a", "handler_a")
    web.get("/b", "handler_b")
    web.get("/c", "handler_c")
    def handler_a(req)
      return web.text("A")
    end
    def handler_b(req)
      return web.text("B")
    end
    def handler_c(req)
      return web.text("C")
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/a").body)
    puts(http.get("http://localhost:#{_port}/b").body)
    puts(http.get("http://localhost:#{_port}/c").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "A")
  test.assert_eq(lines[1], "B")
  test.assert_eq(lines[2], "C")
end

rats "request params hash via dot access"
  script = <<~'SCRIPT'
    use "web"
    use "http"
    web.get("/users/:id/posts/:pid", "nested")
    def nested(req)
      id = req.params["id"]
      pid = req.params["pid"]
      return web.json({"user_id" => id, "post_id" => pid})
    end
    spawn web.listen(0)
    _port = web.port()
    puts(http.get("http://localhost:#{_port}/users/7/posts/42").body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_contains(result["output"], "\"user_id\":\"7\"")
  test.assert_contains(result["output"], "\"post_id\":\"42\"")
end
