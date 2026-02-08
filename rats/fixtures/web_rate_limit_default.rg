# Fixture: rate_limiter default (10 rps) when no web.rate_limit() called
use "web"
use "http"
import "time"

web.middleware("rate_limiter")
web.get("/ping", "ping_handler")

def ping_handler(req)
  return web.text("pong")
end

spawn web.listen(19205)
time.sleep_ms(300)

# Default is 10 rps, all should pass
puts(http.get("http://localhost:19205/ping").body)
puts(http.get("http://localhost:19205/ping").body)
puts(http.get("http://localhost:19205/ping").body)
