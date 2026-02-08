# Fixture: basic web.get with text response
use "web"
use "http"
import "time"

web.get("/hello", "hello_handler")

def hello_handler(req)
  return web.text("Hello, World!")
end

spawn web.listen(19101)
time.sleep_ms(300)

response = http.get("http://localhost:19101/hello").body
puts(response)
