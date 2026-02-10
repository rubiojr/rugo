# Fixture: JSON array response
use "web"
use "http"

web.get("/list", "list_handler")

def list_handler(req)
  return web.json([1, 2, 3])
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/list").body)
