record = {name: "Alice"}
record["rename"] = fn(new_name) record.name = new_name end
record.rename("Bob")
puts record.name
