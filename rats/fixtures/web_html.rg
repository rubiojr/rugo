# Fixture: HTML response
use "web"
use "http"
import "time"

web.get("/page", "page_handler")

def page_handler(req)
  return web.html("<h1>Hello</h1>")
end

spawn web.listen(19106)
time.sleep_ms(300)

puts(http.get("http://localhost:19106/page").body)
