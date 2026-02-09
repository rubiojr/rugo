# Fixture: JSON response
use "web"
use "http"

web.get("/data", "data_handler")

def data_handler(req)
  return web.json({"name" => "rugo", "version" => 1})
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/data").body)
