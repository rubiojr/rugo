# Gummy ORM Example
#
# A simple ORM for Rugo built on the sqlite module.
# Models are defined with hash columns and return smart records
# that can save and delete themselves.

require "github.com/rubiojr/gummy" with db

conn = db.open(":memory:")

# Define models — creates tables automatically
Users = conn.model("users", {name: "text", email: "text", age: "integer"})
Posts = conn.model("posts", {title: "text", body: "text", user_id: "integer"})

# --- Create records ---

alice = Users.insert({name: "Alice", email: "alice@example.com", age: 30})
bob   = Users.insert({name: "Bob", email: "bob@example.com", age: 25})
puts "Created #{alice.name} (id=#{alice.id})"
puts "Created #{bob.name} (id=#{bob.id})"

Posts.insert({title: "Hello World", body: "My first post!", user_id: alice.id})
Posts.insert({title: "Gummy is great", body: "ORMs in Rugo!", user_id: bob.id})

# --- Record CRUD ---

alice.name = "Alicia"
alice.age = 31
alice.save()
puts "Updated: #{alice.name} (#{alice.age})"

# --- Query ---

user = Users.find(1)
puts "Found: #{user.name}"

for u in Users.all()
  puts "  #{u.name} <#{u.email}>"
end

adults = Users.where({"age >=" => 18})
puts "Adults: #{len(adults)}"

user = Users.first({email: "bob@example.com"})
puts "First match: #{user.name}"

# --- Aggregates ---

puts "Total users: #{Users.count(nil)}"
over_25 = Users.count({"age >=" => 25})
puts "Users 25+: #{over_25}"

names = Users.pluck("name")
puts "Names: #{names}"

if Users.exists({email: "bob@example.com"})
  puts "Bob exists!"
end

# --- Lambda iterators ---

Users.each(fn(u)
  puts "  #{u.name} is #{u.age} years old"
end)

emails = Users.map(fn(u) u.email end)
puts "Emails: #{emails}"

# --- Transactions ---

conn.tx(fn()
  Users.insert({name: "Charlie", email: "charlie@example.com", age: 40})
  Users.insert({name: "Diana", email: "diana@example.com", age: 35})
end)
puts "After transaction: #{Users.count(nil)} users"

# --- Bulk operations ---

Users.insert({name: "Kid1", email: "k1@example.com", age: 10})
Users.insert({name: "Kid2", email: "k2@example.com", age: 12})
puts "Before destroy: #{Users.count(nil)} users"
Users.destroy({"age <" => 18})
puts "After destroy: #{Users.count(nil)} users"

# --- Record delete ---

charlie = Users.first({name: "Charlie"})
charlie.delete()
puts "After delete: #{Users.count(nil)} users"

# --- Error handling ---

nobody = try Users.find(999) or nil
if nobody == nil
  puts "Not found: handled gracefully"
end

conn.close()
puts "Done!"
