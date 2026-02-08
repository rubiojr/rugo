BASE_URL = "https://api.example.com"

def helper()
  return "hello"
end

def greet()
  return helper() + " from " + BASE_URL
end
