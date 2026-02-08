import "os"

def make_config(token)
  return {"token" => token}
end

def make_literal()
  return require_typed_lib.make_config("literal-token")
end

def make_from_env()
  t = os.getenv("HOME")
  return require_typed_lib.make_config(t)
end
