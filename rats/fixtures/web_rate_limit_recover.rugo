# Fixture: rate_limiter recovers after waiting
use "web"
use "http"
import "time"

web.rate_limit(2)
web.middleware("rate_limiter")
web.get("/ping", "ping_handler")

def ping_handler(req)
  return web.text("pong")
end

spawn web.listen(0)
_port = web.port()

# Exhaust burst
puts(http.get("http://localhost:#{_port}/ping").body)
puts(http.get("http://localhost:#{_port}/ping").body)
# Rate limited
puts(http.get("http://localhost:#{_port}/ping").body)
# Wait for token replenishment
time.sleep_ms(600)
# Should pass again
puts(http.get("http://localhost:#{_port}/ping").body)
