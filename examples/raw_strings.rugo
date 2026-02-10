# Raw Strings
#
# Single-quoted strings are raw: no escape processing, no interpolation.
# Only \\ (literal backslash) and \' (literal single quote) are recognized.

# Escape sequences stay literal
puts 'hello\nworld'           # prints: hello\nworld (not a newline)
puts 'tab\there'              # prints: tab\there (not a tab)

# Useful for regex patterns, ANSI codes, Windows paths
path = 'C:\Users\name\Documents'
puts path

# Escaped quote and backslash
puts 'it\'s raw'
puts 'back\\slash'

# No interpolation
name = "world"
puts 'hello #{name}'          # prints: hello #{name} (literal)

# Concatenate raw and double-quoted strings
puts 'raw\n' + " escaped\n"

# Compare with double-quoted (which processes escapes)
puts "double: hello\nworld"   # two lines
puts 'single: hello\nworld'   # one line
