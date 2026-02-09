# Fixture: real_ip strips port from remote_addr
use "web"
use "http"

web.middleware("real_ip")
web.get("/ip", "ip_handler")

def ip_handler(req)
  return web.text(req.remote_addr)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/ip").body)
