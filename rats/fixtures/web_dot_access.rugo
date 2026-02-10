# Fixture: dot access on request hash
use "web"
use "http"

web.get("/info", "info_handler")

def info_handler(req)
  return web.text(req.method + " " + req.path)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/info").body)
