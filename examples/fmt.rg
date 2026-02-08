# fmt module - Format strings using Go's fmt.Sprintf/Printf
#
# use "fmt"
#
# Sprintf returns a formatted string:
#   s = fmt.sprintf("Hello %s, you are %d", "Alice", 30)
#
# Printf prints a formatted string to stdout:
#   fmt.printf("%-10s %5d\n", name, score)
#
# Common format verbs:
#   %s  - string
#   %d  - integer
#   %f  - float (%.2f for 2 decimal places)
#   %x  - hex
#   %q  - quoted string
#   %v  - default format
#   %%  - literal percent sign

use "fmt"

name = "World"
age = 30
height = 1.82

# Basic string formatting
greeting = fmt.sprintf("Hello %s!", name)
puts greeting

# Multiple types
info = fmt.sprintf("%s is %d years old and %.2f meters tall", name, age, height)
puts info

# Padded numbers
for i in [1, 42, 100, 7]
  puts fmt.sprintf("  Item %04d", i)
end

# Printf goes directly to stdout
fmt.printf("\n%-12s %6s\n", "Name", "Score")
fmt.printf("%-12s %6d\n", "Alice", 95)
fmt.printf("%-12s %6d\n", "Bob", 87)
fmt.printf("%-12s %6d\n", "Charlie", 92)
