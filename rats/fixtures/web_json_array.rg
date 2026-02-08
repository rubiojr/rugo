# Fixture: JSON array response
use "web"
use "http"
import "time"

web.get("/list", "list_handler")

def list_handler(req)
  return web.json([1, 2, 3])
end

spawn web.listen(19118)
time.sleep_ms(300)

puts(http.get("http://localhost:19118/list"))
