# Fixture: route-level middleware
use "web"
use "http"
import "time"

web.get("/public", "public_handler")
web.get("/private", "private_handler", "require_auth")

def require_auth(req)
  return web.text("unauthorized", 401)
end

def public_handler(req)
  return web.text("public")
end

def private_handler(req)
  return web.text("private")
end

spawn web.listen(19111)
time.sleep_ms(300)

puts(http.get("http://localhost:19111/public").body)
puts(http.get("http://localhost:19111/private").body)
