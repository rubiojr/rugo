# Hyphenated command treated as shell fallback
try docker-compose-nonexistent-xyz --version or err
  puts "caught"
end
puts "subtraction still works"
x = 10
y = 3
puts x - y
