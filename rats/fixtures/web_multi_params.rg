# Fixture: multiple URL parameters
use "web"
use "http"
import "time"

web.get("/posts/:pid/comments/:cid", "show_comment")

def show_comment(req)
  pid = req.params["pid"]
  cid = req.params["cid"]
  return web.text(pid + ":" + cid)
end

spawn web.listen(19103)
time.sleep_ms(300)

puts(http.get("http://localhost:19103/posts/5/comments/99"))
