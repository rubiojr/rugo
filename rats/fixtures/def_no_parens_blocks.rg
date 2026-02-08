use "conv"

def check_positive
  x = 5
  if x > 0
    puts "positive"
  end
end

def count_up
  i = 0
  while i < 3
    i = i + 1
  end
  puts "counted to " + conv.to_s(i)
end

check_positive()
count_up()
