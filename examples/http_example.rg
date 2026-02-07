# HTTP requests in Rugo
#
import "http"

# GET request
response = http.get("https://httpbin.org/get")
puts("GET response:")
puts(response)

# POST request
body = "{\"name\": \"rugo\", \"version\": 1}"
response = http.post("https://httpbin.org/post", body)
puts("POST response:")
puts(response)
