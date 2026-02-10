# Fixture: rate_limiter blocks after burst
use "web"
use "http"

web.rate_limit(2)
web.middleware("rate_limiter")
web.get("/ping", "ping_handler")

def ping_handler(req)
  return web.text("pong")
end

spawn web.listen(0)
_port = web.port()

# First 2 pass (burst = 2)
puts(http.get("http://localhost:#{_port}/ping").body)
puts(http.get("http://localhost:#{_port}/ping").body)
# Third is rate limited
puts(http.get("http://localhost:#{_port}/ping").body)
