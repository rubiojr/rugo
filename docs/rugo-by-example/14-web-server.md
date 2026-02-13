# Web Server

Build HTTP APIs with the `web` module.

```ruby
use "web"

web.get("/", "home")

def home(req)
  return web.text("Hello from Rugo")
end

puts "route registered"
# web.listen(3000)
```

```text
route registered
```

To start the server, uncomment `web.listen(3000)` and run:

```bash
rugo run server.rugo
```

```text
route registered
```
