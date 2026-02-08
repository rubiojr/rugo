# Go Bridge: Access Go stdlib packages directly from Rugo
#
# Use `import` to bridge Go stdlib packages into Rugo.
# Use `use` for Rugo's built-in modules.
# Use `require` for user module files.

import "strings"
import "strconv"
import "math"
import "path/filepath"
use "conv"

# --- strings ---
puts("=== strings ===")
puts(strings.to_upper("hello world"))
puts(strings.contains("hello", "ell"))
puts(strings.replace_all("hello world", "o", "0"))
parts = strings.split("a,b,c", ",")
puts(strings.join(parts, " | "))
puts(strings.trim_space("  trimmed  "))
puts(strings.repeat("go", 3))

# --- strconv ---
puts("")
puts("=== strconv ===")
n = strconv.atoi("42")
puts(conv.to_s(n + 8))
puts(strconv.itoa(100))

# handle errors with try/or
result = try strconv.atoi("not_a_number") or err
  -1
end
puts("parse error fallback: " + conv.to_s(result))

# --- math ---
puts("")
puts("=== math ===")
puts("sqrt(144) = " + conv.to_s(math.sqrt(144.0)))
puts("pow(2,10) = " + conv.to_s(math.pow(2.0, 10.0)))
puts("ceil(3.2) = " + conv.to_s(math.ceil(3.2)))
puts("floor(3.8) = " + conv.to_s(math.floor(3.8)))
puts("round(3.5) = " + conv.to_s(math.round(3.5)))

# --- path/filepath ---
puts("")
puts("=== filepath ===")
puts(filepath.base("/home/user/docs/file.txt"))
puts(filepath.dir("/home/user/docs/file.txt"))
puts(filepath.ext("archive.tar.gz"))
puts(filepath.join("home", "user", "docs"))
puts(filepath.clean("/home/user/../user/./docs"))
