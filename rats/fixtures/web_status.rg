# Fixture: web.status empty response
use "web"
use "http"
import "time"

web.get("/empty", "empty_handler")

def empty_handler(req)
  return web.status(204)
end

spawn web.listen(19115)
time.sleep_ms(300)

# http.get returns a response hash, 204 has empty body
response = http.get("http://localhost:19115/empty").body
puts("body:" + response)
