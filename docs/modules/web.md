# Web Module

A chi-inspired HTTP router for building web servers and REST APIs. Features named-function handlers, dot-accessible request objects, middleware support, URL parameters, route groups, and static file serving.

## Quick Example

```ruby
use "web"

web.get("/", "home")
web.get("/hello/:name", "greet")

def home(req)
  return web.text("Welcome!")
end

def greet(req)
  name = req.params["name"]
  return web.text("Hello, #{name}!")
end

web.listen(3000)
```

## Routes

Register routes with HTTP method helpers. The second argument is the name of the handler function.

```ruby
web.get("/path", "handler")
web.post("/path", "handler")
web.put("/path", "handler")
web.delete("/path", "handler")
web.patch("/path", "handler")
```

### URL Parameters

Use `:name` to capture path segments:

```ruby
web.get("/users/:id", "show_user")
web.get("/posts/:id/comments/:cid", "show_comment")

def show_user(req)
  id = req.params["id"]
  return web.json({"id" => id})
end

def show_comment(req)
  post_id = req.params["id"]
  comment_id = req.params["cid"]
  return web.json({"post" => post_id, "comment" => comment_id})
end
```

## Handlers

Every handler is a regular function that takes a `req` argument and returns a response:

```ruby
def my_handler(req)
  return web.text("Hello!")
end
```

### The Request Object

The `req` parameter is a hash with dot-accessible fields:

| Field | Type | Description |
|-------|------|-------------|
| `req.method` | string | HTTP method (`"GET"`, `"POST"`, etc.) |
| `req.path` | string | Request path (`"/users/42"`) |
| `req.body` | string | Raw request body |
| `req.params` | hash | URL parameters from `:name` segments |
| `req.query` | hash | Query string parameters |
| `req.header` | hash | Request headers |
| `req.remote_addr` | string | Client address |

```ruby
def my_handler(req)
  method = req.method
  id = req.params["id"]
  page = req.query["page"]
  auth = req.header["Authorization"]
  body = req.body
  return web.text("ok")
end
```

## Response Helpers

Response helpers build response objects. Always `return` them from handlers.

### `web.text(body)` / `web.text(body, status)`

Plain text response:

```ruby
def handler(req)
  return web.text("Hello, World!")
end

def not_found(req)
  return web.text("not found", 404)
end
```

### `web.html(body)` / `web.html(body, status)`

HTML response:

```ruby
def handler(req)
  return web.html("<h1>Welcome!</h1>")
end
```

### `web.json(data)` / `web.json(data, status)`

JSON response. Automatically serializes hashes and arrays:

```ruby
def handler(req)
  return web.json({"name" => "Alice", "age" => 30})
end

def created(req)
  return web.json({"id" => 1}, 201)
end
```

### `web.redirect(url)` / `web.redirect(url, status)`

HTTP redirect (defaults to 302):

```ruby
def handler(req)
  return web.redirect("/new-path")
end

def permanent(req)
  return web.redirect("/new-path", 301)
end
```

### `web.status(code)`

Empty response with a status code:

```ruby
def handler(req)
  return web.status(204)
end
```

### Custom Headers

Add headers by setting a `"headers"` key on the response hash:

```ruby
def handler(req)
  resp = web.text("data")
  resp["headers"] = {"X-Custom" => "value"}
  return resp
end
```

## Middleware

Middleware functions intercept requests before they reach handlers. They follow a simple convention:

- Return `nil` → continue to the next middleware/handler
- Return a response → short-circuit and send that response

### Global Middleware

Applied to all routes, in order:

```ruby
web.middleware("logger")
web.middleware("require_auth")
```

### Route-Level Middleware

Pass extra arguments after the handler name:

```ruby
web.get("/admin", "admin_panel", "require_auth", "require_admin")
```

### Built-in Middleware

| Name | Description |
|------|-------------|
| `"logger"` | Logs `METHOD /path remote_addr` to stderr |
| `"recoverer"` | Recovers from handler panics (TODO) |

### Custom Middleware

Write middleware as regular functions:

```ruby
def require_auth(req)
  auth = req.header["Authorization"]
  if auth == nil
    return web.json({"error" => "unauthorized"}, 401)
  end
  return nil
end

def rate_limit(req)
  # return nil to allow the request through
  return nil
end
```

## Route Groups

Group routes under a shared prefix with optional middleware:

```ruby
web.group("/api", "require_auth")
  web.get("/users", "list_users")       # matches /api/users
  web.post("/users", "create_user")     # matches /api/users
  web.get("/users/:id", "show_user")    # matches /api/users/:id
web.end_group()
```

Groups can be nested and middleware stacks compose:

```ruby
web.group("/api")
  web.get("/health", "health")          # no extra middleware
web.end_group()

web.group("/api/admin", "require_auth", "require_admin")
  web.get("/stats", "admin_stats")
web.end_group()
```

## Static Files

Serve files from a directory:

```ruby
web.static("/assets", "./public")
```

A request to `/assets/css/style.css` serves `./public/css/style.css`.

## Server

### `web.listen(port)`

Starts the HTTP server. This call blocks — use `spawn` for background operation:

```ruby
# Blocking (typical for production scripts)
web.listen(3000)

# Background (useful for testing or multi-service scripts)
spawn web.listen(3000)
```

The server logs `web: listening on :PORT` to stderr on startup.

## Full Example

```ruby
use "web"
use "json"

# Middleware
web.middleware("logger")

# Public routes
web.get("/", "home")
web.get("/health", "health")

# API routes with auth
web.group("/api", "require_auth")
  web.get("/users", "list_users")
  web.get("/users/:id", "show_user")
  web.post("/users", "create_user")
web.end_group()

# --- Handlers ---

def home(req)
  return web.html("<h1>Welcome to my API!</h1>")
end

def health(req)
  return web.json({"status" => "ok"})
end

def list_users(req)
  return web.json({"users" => [
    {"id" => 1, "name" => "Alice"},
    {"id" => 2, "name" => "Bob"}
  ]})
end

def show_user(req)
  id = req.params["id"]
  return web.json({"id" => id, "name" => "User #{id}"})
end

def create_user(req)
  data = json.parse(req.body)
  return web.json({"created" => data["name"]}, 201)
end

# --- Middleware ---

def require_auth(req)
  if req.header["Authorization"] == nil
    return web.json({"error" => "unauthorized"}, 401)
  end
  return nil
end

web.listen(3000)
```
