record = {name: "Alice", saved: false}
record["save"] = fn()
  record.saved = true
  return "saved " + record.name
end

puts record.saved
puts record.save()
puts record.saved
