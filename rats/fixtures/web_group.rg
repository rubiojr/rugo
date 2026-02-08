# Fixture: route groups with prefix
use "web"
use "http"
import "time"

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

spawn web.listen(19112)
time.sleep_ms(300)

puts(http.get("http://localhost:19112/"))
puts(http.get("http://localhost:19112/api/users"))
puts(http.get("http://localhost:19112/api/posts"))
