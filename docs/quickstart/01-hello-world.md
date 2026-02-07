# Hello World

Create `hello.rg`:

```ruby
puts "Hello, World!"
```

Run it:

```bash
rugo run hello.rg
```

Or compile to a native binary:

```bash
rugo build hello.rg
./hello
```

`puts` prints a line. `print` does the same without a newline.

```ruby
print "Hello, "
puts "World!"
```

Comments start with `#`:

```ruby
# This is a comment
puts "not a comment"
```

---
Next: [Variables](02-variables.md)
