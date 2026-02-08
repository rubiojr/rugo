# Fixture: multiple HTTP methods
use "web"
use "http"
import "time"

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

spawn web.listen(19114)
time.sleep_ms(300)

puts(http.get("http://localhost:19114/resource").body)
puts(http.post("http://localhost:19114/resource", "").body)
puts(http.put("http://localhost:19114/resource", "").body)
puts(http.delete("http://localhost:19114/resource").body)
puts(http.patch("http://localhost:19114/resource", "").body)
