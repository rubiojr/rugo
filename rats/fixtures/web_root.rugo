# Fixture: root path handler
use "web"
use "http"

web.get("/", "root_handler")

def root_handler(req)
  return web.text("root")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/").body)
