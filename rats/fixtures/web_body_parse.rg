# Fixture: request body in POST handler
use "web"
use "http"
use "json"
import "time"

web.post("/echo", "echo_handler")

def echo_handler(req)
  data = json.parse(req.body)
  return web.json({"received" => data["message"]})
end

spawn web.listen(19116)
time.sleep_ms(300)

response = http.post("http://localhost:19116/echo", "{\"message\":\"hello\"}")
puts(response)
