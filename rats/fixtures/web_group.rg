# Fixture: route groups with prefix
use "web"
use "http"

web.get("/", "home_handler")

web.group("/api")
  web.get("/users", "users_handler")
  web.get("/posts", "posts_handler")
web.end_group()

def home_handler(req)
  return web.text("home")
end

def users_handler(req)
  return web.text("users")
end

def posts_handler(req)
  return web.text("posts")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/").body)
puts(http.get("http://localhost:#{_port}/api/users").body)
puts(http.get("http://localhost:#{_port}/api/posts").body)
