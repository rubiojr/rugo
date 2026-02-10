# Fixture: middleware that passes through
use "web"
use "http"

web.middleware("pass_through")
web.get("/hello", "hello_handler")

def pass_through(req)
  return nil
end

def hello_handler(req)
  return web.text("reached handler")
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/hello").body)
