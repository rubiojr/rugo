# Fixture: rate_limiter blocks after burst
use "web"
use "http"
import "time"

web.rate_limit(2)
web.middleware("rate_limiter")
web.get("/ping", "ping_handler")

def ping_handler(req)
  return web.text("pong")
end

spawn web.listen(19203)
time.sleep_ms(300)

# First 2 pass (burst = 2)
puts(http.get("http://localhost:19203/ping"))
puts(http.get("http://localhost:19203/ping"))
# Third is rate limited
puts(http.get("http://localhost:19203/ping"))
