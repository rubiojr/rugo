# HTTP requests in Rugo
#
# All http methods return a response hash with:
#   status_code - HTTP status code (integer)
#   body        - Response body (string)
#   headers     - Response headers (hash)
#
use "http"

# GET request
resp = http.get("https://httpbin.org/get")
puts("Status: " + resp.status_code)
puts("Body: " + resp.body)

# POST request
body = "{\"name\": \"rugo\", \"version\": 1}"
resp = http.post("https://httpbin.org/post", body)
puts("POST status: " + resp.status_code)
puts("POST body: " + resp.body)

# GET with custom headers
headers = {"Authorization" => "Bearer my-token", "Accept" => "application/json"}
resp = http.get("https://httpbin.org/headers", headers)
puts("Headers response: " + resp.body)

# PUT request
resp = http.put("https://httpbin.org/put", "{\"updated\": true}")
puts("PUT status: " + resp.status_code)

# PATCH request
resp = http.patch("https://httpbin.org/patch", "{\"title\": \"new title\"}")
puts("PATCH status: " + resp.status_code)

# DELETE request
resp = http.delete("https://httpbin.org/delete")
puts("DELETE status: " + resp.status_code)

# Error checking via status code
resp = http.get("https://httpbin.org/status/404")
if resp.status_code != 200
  puts("Error: got status " + resp.status_code)
end
