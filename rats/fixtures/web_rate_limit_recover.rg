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

spawn web.listen(19204)
time.sleep_ms(300)

# Exhaust burst
puts(http.get("http://localhost:19204/ping"))
puts(http.get("http://localhost:19204/ping"))
# Rate limited
puts(http.get("http://localhost:19204/ping"))
# Wait for token replenishment
time.sleep_ms(600)
# Should pass again
puts(http.get("http://localhost:19204/ping"))
