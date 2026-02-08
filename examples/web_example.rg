# Web Module Example — Simple REST API
#
# Run with: rugo run examples/web_example.rg
# Then visit http://localhost:3000 in your browser
# or test with curl:
#   curl http://localhost:3000/
#   curl http://localhost:3000/health
#   curl http://localhost:3000/api/users
#   curl http://localhost:3000/api/users/42
#   curl -X POST http://localhost:3000/api/users -d '{"name":"Alice"}'

use "web"
use "json"

# --- Middleware ---

web.middleware("logger")

# --- Public routes ---

web.get("/", "home")
web.get("/health", "health")

# --- API routes (with auth middleware) ---

web.group("/api", "require_auth")
  web.get("/users", "list_users")
  web.get("/users/:id", "show_user")
  web.post("/users", "create_user")
web.end_group()

# --- Handlers ---

def home(req)
  return web.html(<<~HTML
    <h1>Welcome to Rugo Web!</h1>
    <p>Try these endpoints:</p>
    <ul>
      <li><a href="/health">/health</a></li>
      <li><a href="/api/users">/api/users</a> (needs Authorization header)</li>
    </ul>
  HTML)
end

def health(req)
  return web.json({"status" => "ok"})
end

def list_users(req)
  users = [
    {"id" => 1, "name" => "Alice"},
    {"id" => 2, "name" => "Bob"},
    {"id" => 3, "name" => "Charlie"}
  ]
  return web.json({"users" => users})
end

def show_user(req)
  id = req.params["id"]
  return web.json({"id" => id, "name" => "User #{id}"})
end

def create_user(req)
  data = json.parse(req.body)
  return web.json({"created" => data["name"]}, 201)
end

# --- Custom middleware ---

def require_auth(req)
  auth = req.header["Authorization"]
  if auth == nil
    return web.json({"error" => "unauthorized"}, 401)
  end
  return nil
end

# --- Start server ---

puts("Starting server on http://localhost:3000")
web.listen(3000)
