# Fixture: real_ip strips port from remote_addr
use "web"
use "http"
import "time"

web.middleware("real_ip")
web.get("/ip", "ip_handler")

def ip_handler(req)
  return web.text(req.remote_addr)
end

spawn web.listen(19201)
time.sleep_ms(300)

puts(http.get("http://localhost:19201/ip").body)
