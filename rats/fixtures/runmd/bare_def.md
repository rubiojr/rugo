# Fixture: bare def (syntax error)

```ruby
def greet(name)
  puts "Hello, #{name}!"
end

def
greet("World")

scores = [90, 85, 72]
for score in scores
  if score >= 90
    puts "#{score} → A"
  else
    puts "#{score} → B"
  end
end
```
