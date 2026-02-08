# Fixture: query string parameters
use "web"
use "http"
import "time"

web.get("/search", "search_handler")

def search_handler(req)
  q = req.query["q"]
  limit = req.query["limit"]
  return web.text(q + ":" + limit)
end

spawn web.listen(19107)
time.sleep_ms(300)

puts(http.get("http://localhost:19107/search?q=rugo&limit=10"))
