# Fixture: JSON response
use "web"
use "http"
import "time"

web.get("/data", "data_handler")

def data_handler(req)
  return web.json({"name" => "rugo", "version" => 1})
end

spawn web.listen(19104)
time.sleep_ms(300)

puts(http.get("http://localhost:19104/data").body)
