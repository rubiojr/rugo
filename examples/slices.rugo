# Array Slicing in Rugo
use "conv"

numbers = [10, 20, 30, 40, 50, 60, 70, 80, 90, 100]

# Take the first 3 elements
first_three = numbers[0, 3]
puts("First three: " + conv.to_s(first_three))

# Take 4 elements starting at index 2
middle = numbers[2, 4]
puts("Middle four: " + conv.to_s(middle))

# Slice past the end is clamped silently
tail = numbers[7, 10]
puts("Tail (clamped): " + conv.to_s(tail))

# Start past the end returns an empty array
empty = numbers[100, 5]
puts("Empty: " + conv.to_s(empty))
puts("Empty length: " + conv.to_s(len(empty)))

# Practical use: paginate a list
items = ["a", "b", "c", "d", "e", "f", "g", "h"]
page_size = 3
page1 = items[0, page_size]
page2 = items[3, page_size]
page3 = items[6, page_size]
puts("Page 1: " + conv.to_s(page1))
puts("Page 2: " + conv.to_s(page2))
puts("Page 3: " + conv.to_s(page3))
