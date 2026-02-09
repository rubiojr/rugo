# Fixture: JSON with status code
use "web"
use "http"

web.post("/items", "create_item")

def create_item(req)
  return web.json({"created" => true, "body" => req.body}, 201)
end

spawn web.listen(0)
_port = web.port()

puts(http.post("http://localhost:#{_port}/items", "{\"name\":\"test\"}").body)
