# Fixture: query string parameters
use "web"
use "http"

web.get("/search", "search_handler")

def search_handler(req)
  q = req.query["q"]
  limit = req.query["limit"]
  return web.text(q + ":" + limit)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/search?q=rugo&limit=10").body)
