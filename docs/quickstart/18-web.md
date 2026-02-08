# Web Server

Build web servers and REST APIs with the `web` module.

## Hello World

```ruby
use "web"

web.get("/", "home")

def home(req)
  return web.text("Hello, World!")
end

web.listen(3000)
```

Run it and visit `http://localhost:3000`:

```bash
rugo run server.rg
```

## Routes and URL Parameters

Register routes with HTTP method helpers. Use `:name` to capture path segments:

```ruby
use "web"

web.get("/", "home")
web.get("/users/:id", "show_user")
web.post("/users", "create_user")

def home(req)
  return web.text("Welcome!")
end

def show_user(req)
  id = req.params["id"]
  return web.json({"id" => id})
end

def create_user(req)
  return web.json({"created" => true}, 201)
end

web.listen(3000)
```

All five HTTP methods are supported: `web.get`, `web.post`, `web.put`, `web.delete`, `web.patch`.

## The Request Object

Every handler receives a `req` hash with dot-accessible fields:

```ruby
def my_handler(req)
  req.method        # "GET", "POST", etc.
  req.path          # "/users/42"
  req.body          # raw request body
  req.params["id"]  # URL parameters
  req.query["page"] # query string parameters
  req.header["Authorization"]  # request headers
  req.remote_addr   # client address
end
```

## Response Helpers

```ruby
web.text("hello")                    # 200 text/plain
web.text("not found", 404)           # 404 text/plain
web.html("<h1>Hi</h1>")             # 200 text/html
web.json({"key" => "val"})          # 200 application/json
web.json({"key" => "val"}, 201)     # with status code
web.redirect("/login")              # 302 redirect
web.redirect("/new", 301)           # 301 permanent
web.status(204)                     # empty response
```

## Middleware

Middleware functions intercept requests. Return `nil` to continue, or a response to stop:

```ruby
use "web"

web.middleware("require_auth")
web.get("/secret", "secret_handler")

def require_auth(req)
  if req.header["Authorization"] == nil
    return web.json({"error" => "unauthorized"}, 401)
  end
  return nil
end

def secret_handler(req)
  return web.text("secret data")
end

web.listen(3000)
```

Built-in middleware: `"logger"`, `"real_ip"`, `"rate_limiter"`.

### Real IP

Resolves the real client IP from proxy headers (`X-Forwarded-For`, `X-Real-Ip`):

```ruby
web.middleware("real_ip")
web.middleware("logger")    # logger now shows the real IP
```

### Rate Limiter

Per-IP rate limiting with token bucket algorithm:

```ruby
web.rate_limit(100)              # 100 requests/second per IP
web.middleware("rate_limiter")   # returns 429 when exceeded
```

Route-level middleware (extra arguments after handler name):

```ruby
web.get("/admin", "admin_panel", "require_auth", "require_admin")
```

Built-in middleware: `"logger"` logs requests to stderr.

## Route Groups

Group routes under a shared prefix with optional middleware:

```ruby
web.group("/api", "require_auth")
  web.get("/users", "list_users")
  web.post("/users", "create_user")
web.end_group()
```

## Next Steps

See the full [web module reference](../modules/web.md) for static file serving, custom headers, and more examples.

---

[← Structs](17-structs.md)
