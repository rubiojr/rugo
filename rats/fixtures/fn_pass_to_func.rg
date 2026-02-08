def my_map(f, arr)
  result = []
  for item in arr
    result = append(result, f(item))
  end
  return result
end

nums = my_map(fn(x) x * 2 end, [1, 2, 3])
puts nums[0]
puts nums[1]
puts nums[2]
