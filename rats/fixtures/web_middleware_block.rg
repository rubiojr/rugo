# Fixture: global middleware
use "web"
use "http"
import "time"

web.middleware("block_all")
web.get("/hello", "hello_handler")

def block_all(req)
  return web.text("blocked", 403)
end

def hello_handler(req)
  return web.text("should not reach")
end

spawn web.listen(19109)
time.sleep_ms(300)

puts(http.get("http://localhost:19109/hello"))
