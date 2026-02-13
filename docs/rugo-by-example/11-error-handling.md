# Error Handling

Use `try` / `or` to recover from failures.

```ruby
hostname = try `echo localhost` or "localhost"
puts hostname
```

```text
localhost
```

Block form gives access to the error:

```ruby
data = try `cat /missing-file 2>/dev/null` or err
  puts "Error happened"
  "fallback"
end

puts data
```

```text
Error happened
fallback
```
