# Hello World

Create `hello.rugo`:

```ruby
puts "Hello, World!"
```

Run it:

```bash
rugo run hello.rugo
```

Or compile to a native binary:

```bash
rugo build hello.rugo
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
