# Fixture: multiple URL parameters
use "web"
use "http"

web.get("/posts/:pid/comments/:cid", "show_comment")

def show_comment(req)
  pid = req.params["pid"]
  cid = req.params["cid"]
  return web.text(pid + ":" + cid)
end

spawn web.listen(0)
_port = web.port()

puts(http.get("http://localhost:#{_port}/posts/5/comments/99").body)
