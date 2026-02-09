# RATS: HTTP module — response hash, custom headers, PUT/PATCH/DELETE
use "test"
use "str"
use "web"

# --- Response hash structure ---

rats "http.get returns response hash with status_code, body, headers"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.get("/info", "info_handler")
    def info_handler(req)
      return web.json({"msg" => "ok"})
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.get("http://localhost:#{p}/info")
    puts(resp.status_code)
    puts(resp.body)
    puts(resp.headers["Content-Type"])
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_contains(lines[1], "msg")
  test.assert_contains(lines[2], "application/json")
end

rats "http.post returns response hash"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.post("/echo", "echo_handler")
    def echo_handler(req)
      return web.text("got it")
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.post("http://localhost:#{p}/echo", "payload")
    puts(resp.status_code)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_eq(lines[1], "got it")
end

rats "http.get non-200 status code accessible"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.get("/missing", "missing_handler")
    def missing_handler(req)
      return web.status(404)
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.get("http://localhost:#{p}/missing")
    puts(resp.status_code)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "404")
end

# --- Custom headers ---

rats "http.get sends custom headers"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.get("/check", "check_handler")
    def check_handler(req)
      return web.text(req.header["X-Custom"])
    end
    spawn web.listen(#{p})
    web.port()
    headers = {"X-Custom" => "hello-rugo"}
    resp = http.get("http://localhost:#{p}/check", headers)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "hello-rugo")
end

rats "http.post sends custom headers"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.post("/check", "check_handler")
    def check_handler(req)
      return web.text(req.header["Authorization"])
    end
    spawn web.listen(#{p})
    web.port()
    headers = {"Authorization" => "Bearer token123"}
    resp = http.post("http://localhost:#{p}/check", "{}", headers)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Bearer token123")
end

# --- PUT, PATCH, DELETE methods ---

rats "http.put sends PUT request"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.put("/resource", "put_handler")
    def put_handler(req)
      return web.text("PUT:" + req.body)
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.put("http://localhost:#{p}/resource", "update-data")
    puts(resp.status_code)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_eq(lines[1], "PUT:update-data")
end

rats "http.patch sends PATCH request"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.patch("/resource", "patch_handler")
    def patch_handler(req)
      return web.text("PATCH:" + req.body)
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.patch("http://localhost:#{p}/resource", "partial-data")
    puts(resp.status_code)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_eq(lines[1], "PATCH:partial-data")
end

rats "http.delete sends DELETE request"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.delete("/resource", "delete_handler")
    def delete_handler(req)
      return web.text("DELETED")
    end
    spawn web.listen(#{p})
    web.port()
    resp = http.delete("http://localhost:#{p}/resource")
    puts(resp.status_code)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  lines = result["lines"]
  test.assert_eq(lines[0], "200")
  test.assert_eq(lines[1], "DELETED")
end

rats "http.delete sends custom headers"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.delete("/auth", "auth_handler")
    def auth_handler(req)
      return web.text(req.header["Authorization"])
    end
    spawn web.listen(#{p})
    web.port()
    headers = {"Authorization" => "Bearer secret"}
    resp = http.delete("http://localhost:#{p}/auth", headers)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "Bearer secret")
end

rats "http.put sends custom headers"
  p = web.free_port()
  script = <<~SCRIPT
    use "web"
    use "http"
    web.put("/auth", "auth_handler")
    def auth_handler(req)
      return web.text(req.header["X-Api-Key"])
    end
    spawn web.listen(#{p})
    web.port()
    headers = {"X-Api-Key" => "key123"}
    resp = http.put("http://localhost:#{p}/auth", "{}", headers)
    puts(resp.body)
  SCRIPT
  test.write_file(test.tmpdir() + "/test.rg", script)
  result = test.run("rugo run " + test.tmpdir() + "/test.rg")
  test.assert_eq(result["status"], 0)
  test.assert_eq(result["output"], "key123")
end
