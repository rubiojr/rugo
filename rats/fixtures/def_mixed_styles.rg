def no_parens
  puts "no-parens"
end

def empty_parens()
  puts "empty-parens"
end

def with_params(msg)
  puts "with-params: " + msg
end

no_parens()
empty_parens()
with_params("hello")
