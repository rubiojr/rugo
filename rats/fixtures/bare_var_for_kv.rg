# Fixture: for k, v in hash — both variables tracked
h = {"a" => 1, "b" => 2}
keys = []
vals = []
for k, v in h
  keys = append(keys, k)
  vals = append(vals, v)
end
puts len(keys)
puts len(vals)
