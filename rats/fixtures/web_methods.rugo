# Fixture: multiple HTTP methods
use "web"
use "http"

web.get("/resource", "get_resource")
web.post("/resource", "post_resource")
web.put("/resource", "put_resource")
web.delete("/resource", "delete_resource")
web.patch("/resource", "patch_resource")

def get_resource(req)
  return web.text("GET")
end

def post_resource(req)
  return web.text("POST")
end

def put_resource(req)
  return web.text("PUT")
end

def delete_resource(req)
  return web.text("DELETE")
end

def patch_resource(req)
  return web.text("PATCH")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/resource").body)
puts(http.post("http://localhost:#{_port}/resource", "").body)
puts(http.put("http://localhost:#{_port}/resource", "").body)
puts(http.delete("http://localhost:#{_port}/resource").body)
puts(http.patch("http://localhost:#{_port}/resource", "").body)
