# Fixture: web.status empty response
use "web"
use "http"

web.get("/empty", "empty_handler")

def empty_handler(req)
  return web.status(204)
end

spawn web.listen(0)
_port = web.port()

# http.get returns a response hash, 204 has empty body
response = http.get("http://localhost:#{_port}/empty").body
puts("body:" + response)
