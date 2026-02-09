# Fixture: rate_limiter default (10 rps) when no web.rate_limit() called
use "web"
use "http"

web.middleware("rate_limiter")
web.get("/ping", "ping_handler")

def ping_handler(req)
  return web.text("pong")
end

spawn web.listen(0)
_port = web.port()

# Default is 10 rps, all should pass
puts(http.get("http://localhost:#{_port}/ping").body)
puts(http.get("http://localhost:#{_port}/ping").body)
puts(http.get("http://localhost:#{_port}/ping").body)
