# Fixture: HTML response
use "web"
use "http"

web.get("/page", "page_handler")

def page_handler(req)
  return web.html("<h1>Hello</h1>")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/page").body)
