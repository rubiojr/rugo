# Arrays and Hashes in Rugo
import "conv"

# Arrays
fruits = ["apple", "banana", "cherry"]
puts("Fruits:")
puts(fruits)
puts("Length: " + conv.to_s(len(fruits)))
puts("First: " + fruits[0])
puts("Last: " + fruits[2])

# Array append
fruits = append(fruits, "date")
puts("After append: " + conv.to_s(len(fruits)))

# Hashes
person = {"name" => "Alice", "age" => 30, "city" => "NYC"}
puts("Person:")
puts(person)
puts("Name: " + person["name"])

# Numbers in arrays
numbers = [10, 20, 30, 40, 50]
sum = 0
i = 0
while i < len(numbers)
  sum = sum + numbers[i]
  i = i + 1
end
puts("Sum: " + conv.to_s(sum))
