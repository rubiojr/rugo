# re module - Regular expressions
#
# use "re"
#
# All functions take the pattern as the first argument.
# Invalid patterns panic (use try/or to catch).

use "re"

# Test if a pattern matches
email = "user@example.com"
if re.test("\\w+@\\w+\\.\\w+", email)
  puts "Valid email: " + email
end

# Find first match
text = "Order #12345 shipped on 2024-01-15"
order_num = re.find("#(\\d+)", text)
puts "Found: " + order_num

# Find all matches
numbers = re.find_all("\\d+", text)
puts "All numbers: "
for n in numbers
  puts "  " + n
end

# Replace first occurrence
puts re.replace("\\d+", "version 1.2.3", "X")

# Replace all occurrences
puts re.replace_all("\\d+", "a1b2c3", "*")

# Split by pattern
parts = re.split(",\\s*", "apple, banana,cherry,  date")
for p in parts
  puts "  - " + p
end

# Match with capture groups
m = re.match("(\\w+)@(\\w+)\\.(\\w+)", email)
if m != nil
  puts "Full match: " + m["match"]
  groups = m["groups"]
  puts "User: " + groups[0]
  puts "Domain: " + groups[1]
  puts "TLD: " + groups[2]
end

# Error handling for invalid patterns
result = try re.test("[invalid", "test") or "bad pattern"
puts result
