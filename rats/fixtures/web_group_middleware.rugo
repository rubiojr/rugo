# Fixture: route group with middleware
use "web"
use "http"

web.get("/public", "public_handler")

web.group("/admin", "block_admin")
  web.get("/dashboard", "dashboard_handler")
web.end_group()

def block_admin(req)
  return web.text("forbidden", 403)
end

def public_handler(req)
  return web.text("public")
end

def dashboard_handler(req)
  return web.text("dashboard")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/public").body)
puts(http.get("http://localhost:#{_port}/admin/dashboard").body)
