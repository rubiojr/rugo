# Fixture: route group with middleware
use "web"
use "http"
import "time"

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

spawn web.listen(19113)
time.sleep_ms(300)

puts(http.get("http://localhost:19113/public").body)
puts(http.get("http://localhost:19113/admin/dashboard").body)
