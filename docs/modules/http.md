# http

HTTP client module. All methods return a **response hash** with:

| Key | Type | Description |
|-----|------|-------------|
| `status_code` | int | HTTP status code (200, 404, etc.) |
| `body` | string | Response body |
| `headers` | hash | Response headers (`{"Content-Type" => "application/json", ...}`) |

```ruby
use "http"
```

## get

Performs an HTTP GET request. Returns a response hash.

```ruby
resp = http.get("https://httpbin.org/get")
puts resp.status_code  # 200
puts resp.body         # response body
puts resp.headers["Content-Type"]
```

Optional second argument sets custom headers:

```ruby
headers = {"Authorization" => "Bearer token123", "Accept" => "application/json"}
resp = http.get("https://api.example.com/data", headers)
```

Panics on network errors.

## post

Performs an HTTP POST request. Returns a response hash.

```ruby
resp = http.post("https://httpbin.org/post", "{\"name\": \"rugo\"}")
puts resp.status_code
puts resp.body
```

Optional third argument sets custom headers (default Content-Type is `application/json`):

```ruby
headers = {"Authorization" => "Bearer token123"}
resp = http.post(url, body, headers)
```

Panics on network errors.

## put

Performs an HTTP PUT request. Returns a response hash.

```ruby
resp = http.put(url, "{\"title\": \"updated\"}", headers)
puts resp.status_code
```

Optional third argument sets custom headers.

## patch

Performs an HTTP PATCH request. Returns a response hash.

```ruby
resp = http.patch(url, "{\"title\": \"new title\"}", headers)
puts resp.status_code
```

Optional third argument sets custom headers.

## delete

Performs an HTTP DELETE request. Returns a response hash.

```ruby
resp = http.delete(url)
puts resp.status_code
```

Optional second argument sets custom headers:

```ruby
headers = {"Authorization" => "Bearer token123"}
resp = http.delete(url, headers)
```

Panics on network errors.

## Error Handling

HTTP methods do **not** panic on non-2xx status codes — check `status_code` instead:

```ruby
resp = http.get("https://api.example.com/missing")
if resp.status_code != 200
  puts "Error: " + resp.body
end
```

Network errors (connection refused, DNS failure) still panic and integrate with `try/or`:

```ruby
resp = try http.get("http://unreachable:9999") or nil
if resp == nil
  puts "request failed"
end
```
