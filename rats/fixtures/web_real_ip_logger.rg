# Fixture: real_ip + logger ordering (real_ip before logger shows correct IP)
use "web"
use "http"
import "time"

web.middleware("real_ip")
web.middleware("logger")
web.get("/check", "check_handler")

def check_handler(req)
  return web.text(req.remote_addr)
end

spawn web.listen(19206)
time.sleep_ms(300)

puts(http.get("http://localhost:19206/check"))
