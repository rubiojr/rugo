# Fixture: JSON with status code
use "web"
use "http"
import "time"

web.post("/items", "create_item")

def create_item(req)
  return web.json({"created" => true, "body" => req.body}, 201)
end

spawn web.listen(19105)
time.sleep_ms(300)

puts(http.post("http://localhost:19105/items", "{\"name\":\"test\"}"))
