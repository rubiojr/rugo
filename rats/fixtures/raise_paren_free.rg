result = try raise "kaboom" or err
  "caught: " + err
end
puts result
