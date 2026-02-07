# Slug module example
# Build with: cd custom-rugo && go build -o myrugo . && ./myrugo ../example.rg

import "slug"

title = "Hello, World! This is Rugo"
puts slug.make(title)

puts slug.make_lang("Cześć świat", "pl")

puts slug.is_slug("hello-world")
puts slug.is_slug("Hello World!")

puts slug.join("custom", "rugo", "modules")
