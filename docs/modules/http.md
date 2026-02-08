# http

HTTP client module.

```ruby
use "http"
```

## get

Performs an HTTP GET request. Returns the response body as a string.

```ruby
body = http.get("https://httpbin.org/get")
puts body
```

Panics on network errors.

## post

Performs an HTTP POST request. Returns the response body as a string.

```ruby
response = http.post("https://httpbin.org/post", "payload")
```

Optional third argument sets the content type (defaults to `application/json`):

```ruby
response = http.post(url, "<h1>hi</h1>", "text/html")
```

Panics on network errors.
