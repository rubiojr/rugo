# Fixture: request body in POST handler
use "web"
use "http"
use "json"

web.post("/echo", "echo_handler")

def echo_handler(req)
  data = json.parse(req.body)
  return web.json({"received" => data["message"]})
end

spawn web.listen(0)
_port = web.port()

response = http.post("http://localhost:#{_port}/echo", "{\"message\":\"hello\"}").body
puts(response)
