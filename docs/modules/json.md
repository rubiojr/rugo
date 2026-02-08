# json

JSON parsing and encoding.

```ruby
use "json"
```

## parse

Parses a JSON string into Rugo values (hashes, arrays, strings, numbers, booleans, nil).
JSON objects become hashes, JSON arrays become arrays, and whole numbers become integers.

```ruby
data = json.parse("{\"name\": \"rugo\", \"version\": 1}")
puts data["name"]      # rugo
puts data["version"]   # 1

arr = json.parse("[1, 2, 3]")
puts arr[0]   # 1
puts len(arr) # 3
```

Panics on invalid JSON.

## encode

Converts a Rugo value (hash, array, string, number, boolean, nil) to a JSON string.

```ruby
data = {"name" => "rugo", "version" => 1}
puts json.encode(data)   # {"name":"rugo","version":1}

arr = [1, "two", true]
puts json.encode(arr)     # [1,"two",true]
```

## Example: Fetching and parsing an API

```ruby
use "http"
use "json"

body = http.get("https://api.example.com/data.json")
data = json.parse(body)
puts data["title"]
```
