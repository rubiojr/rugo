# Pipe: user-defined function in pipe chain
def double(x)
  return x + " " + x
end

echo "hello" | double | puts
