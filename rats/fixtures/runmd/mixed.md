# Fixture: mix of passing and failing Rugo snippets

Valid Rugo:

```ruby
puts "hello"
```

Broken Rugo (unknown module):

```ruby
use "nonexistent_module_xyz"
puts "hello"
```

Another valid Rugo:

```ruby
x = 1
puts x
```
