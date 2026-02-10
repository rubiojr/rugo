# Fixture: route-level middleware
use "web"
use "http"

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

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/public").body)
puts(http.get("http://localhost:#{_port}/private").body)
