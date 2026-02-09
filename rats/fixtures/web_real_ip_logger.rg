# Fixture: real_ip + logger ordering (real_ip before logger shows correct IP)
use "web"
use "http"

web.middleware("real_ip")
web.middleware("logger")
web.get("/check", "check_handler")

def check_handler(req)
  return web.text(req.remote_addr)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/check").body)
