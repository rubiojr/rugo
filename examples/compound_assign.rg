# Compound assignment operators in Rugo
import "conv"

# Basic operators
x = 10
puts "start: #{x}"

x += 5
puts "x += 5: #{x}"

x -= 3
puts "x -= 3: #{x}"

x *= 2
puts "x *= 2: #{x}"

x /= 4
puts "x /= 4: #{x}"

x %= 4
puts "x modulo 4: #{x}"

# String building
result = ""
result += "Hello"
result += ", "
result += "World!"
puts result

# Accumulator in a loop
numbers = [10, 20, 30, 40, 50]
sum = 0
i = 0
while i < len(numbers)
  sum += numbers[i]
  i += 1
end
puts "Sum: #{sum}"
