# Heredoc strings — multiline string literals
#
# Rugo supports heredoc syntax for multiline strings,
# with and without string interpolation.

# Interpolating heredoc — like double-quoted strings, #{} works
name = "World"
greeting = <<HTML
<h1>Hello #{name}</h1>
<p>Welcome to Rugo!</p>
HTML
puts greeting

# Squiggly heredoc (<<~) — strips common leading whitespace
# Great for keeping your code indented cleanly
site = "rugo.dev"
page = <<~HTML
  <html>
    <body>
      <h1>#{name}</h1>
      <p>Visit #{site}</p>
    </body>
  </html>
HTML
puts page

# Raw heredoc — no interpolation, content is literal
template = <<'CODE'
def #{method_name}(#{args})
  puts "generated code"
end
CODE
puts template

# Raw squiggly heredoc — literal content + indent stripping
snippet = <<~'YAML'
  name: myapp
  version: 1.0
  tags:
    - production
    - #{not_interpolated}
YAML
puts snippet
