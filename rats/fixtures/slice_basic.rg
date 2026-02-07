arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]

# First 3 elements
first = arr[0, 3]
puts(len(first))
puts(first[0])
puts(first[2])

# Middle slice
mid = arr[3, 4]
puts(len(mid))
puts(mid[0])

# Slice beyond bounds (clamp)
big = arr[8, 10]
puts(len(big))

# Start beyond bounds (empty)
empty = arr[100, 5]
puts(len(empty))
