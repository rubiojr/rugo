# Fixture: global middleware
use "web"
use "http"

web.middleware("block_all")
web.get("/hello", "hello_handler")

def block_all(req)
  return web.text("blocked", 403)
end

def hello_handler(req)
  return web.text("should not reach")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/hello").body)
