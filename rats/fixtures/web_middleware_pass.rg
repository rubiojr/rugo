# Fixture: middleware that passes through
use "web"
use "http"
import "time"

web.middleware("pass_through")
web.get("/hello", "hello_handler")

def pass_through(req)
  return nil
end

def hello_handler(req)
  return web.text("reached handler")
end

spawn web.listen(19110)
time.sleep_ms(300)

puts(http.get("http://localhost:19110/hello").body)
