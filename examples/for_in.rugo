# for..in loops, break, and next in Rugo

# Simple array iteration
fruits = ["apple", "banana", "cherry"]
puts "=== Simple iteration ==="
for fruit in fruits
  puts fruit
end

# With index (two-var form: index, value)
puts ""
puts "=== With index ==="
for i, fruit in fruits
  puts "#{i}: #{fruit}"
end

# Hash iteration (two-var form: key, value)
puts ""
puts "=== Hash iteration ==="
person = {"name" => "Alice", "age" => 30, "city" => "NYC"}
for key, value in person
  puts "#{key} => #{value}"
end

# Summing with for..in and +=
puts ""
puts "=== Sum ==="
numbers = [10, 20, 30, 40, 50]
sum = 0
for n in numbers
  sum += n
end
puts "Sum: #{sum}"

# Nested iteration
puts ""
puts "=== Nested ==="
matrix = [[1, 2, 3], [4, 5, 6]]
for row in matrix
  for val in row
    print "#{val} "
  end
  puts ""
end

# Break — stop at first match
puts ""
puts "=== Break ==="
items = ["a", "b", "STOP", "c", "d"]
for item in items
  if item == "STOP"
    break
  end
  puts item
end

# Next — skip odd numbers
puts ""
puts "=== Next ==="
nums = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
for n in nums
  if n % 2 != 0
    next
  end
  puts n
end

# Break in while
puts ""
puts "=== While + break ==="
count = 0
while true
  count += 1
  if count > 3
    break
  end
  puts "count: #{count}"
end
puts "done"
