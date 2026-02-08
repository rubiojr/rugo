# Fixture: URL parameters
use "web"
use "http"
import "time"

web.get("/users/:id", "show_user")

def show_user(req)
  id = req.params["id"]
  return web.text("user:" + id)
end

spawn web.listen(19102)
time.sleep_ms(300)

puts(http.get("http://localhost:19102/users/42"))
puts(http.get("http://localhost:19102/users/hello"))
