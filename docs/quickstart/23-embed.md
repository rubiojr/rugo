# File Embedding

Embed file contents into your binary at compile time with `embed`:

```ruby
embed "config.yaml" as config
puts config
```

The file is read during compilation and baked into the executable. No external files needed at runtime.

## Multiple embeds

```ruby
embed "header.html" as header
embed "footer.html" as footer

puts header + content + footer
```

## Path rules

Paths are relative to the source file. Files must be in the same directory or a subdirectory — you cannot use `../` to escape:

```ruby
embed "assets/logo.txt" as logo     # OK
embed "../secret.txt" as secret     # ERROR: escapes source directory
```

This mirrors Go's `embed` restriction and prevents libraries from accessing files outside their own directory tree.
