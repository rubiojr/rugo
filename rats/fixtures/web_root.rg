# Fixture: root path handler
use "web"
use "http"
import "time"

web.get("/", "root_handler")

def root_handler(req)
  return web.text("root")
end

spawn web.listen(19117)
time.sleep_ms(300)

puts(http.get("http://localhost:19117/"))
