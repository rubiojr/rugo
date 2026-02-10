# Fixture: URL parameters
use "web"
use "http"

web.get("/users/:id", "show_user")

def show_user(req)
  id = req.params["id"]
  return web.text("user:" + id)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/users/42").body)
puts(http.get("http://localhost:#{_port}/users/hello").body)
