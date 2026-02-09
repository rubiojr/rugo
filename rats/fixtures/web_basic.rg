# Fixture: basic web.get with text response
use "web"
use "http"

web.get("/hello", "hello_handler")

def hello_handler(req)
  return web.text("Hello, World!")
end

spawn web.listen(0)
_port = web.port()

response = http.get("http://localhost:#{_port}/hello").body
puts(response)
