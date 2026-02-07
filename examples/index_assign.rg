# Index assignment for arrays and hashes

# Array element mutation
puts "=== Array mutation ==="
fruits = ["apple", "banana", "cherry"]
puts fruits
fruits[1] = "blueberry"
puts fruits

# Hash entry mutation and addition
puts ""
puts "=== Hash mutation ==="
config = {"host" => "localhost", "port" => 3000}
puts config
config["port"] = 8080
config["debug"] = true
puts config

# Building a hash incrementally
puts ""
puts "=== Word count ==="
counts = {}
words = ["hello", "world", "hello", "hello", "world"]
for word in words
  if counts[word]
    counts[word] += 1
  else
    counts[word] = 1
  end
end
puts counts

# Swapping array elements
puts ""
puts "=== Swap ==="
arr = [1, 2, 3, 4, 5]
puts arr
tmp = arr[0]
arr[0] = arr[4]
arr[4] = tmp
puts arr
