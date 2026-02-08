# Fixture: dot access on request hash
use "web"
use "http"
import "time"

web.get("/info", "info_handler")

def info_handler(req)
  return web.text(req.method + " " + req.path)
end

spawn web.listen(19108)
time.sleep_ms(300)

puts(http.get("http://localhost:19108/info"))
