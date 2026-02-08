# Collections benchmarks
# Tracks: array/hash creation, iteration, index access, rugo_iterable overhead
use "bench"

bench "array creation (10 elements)"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
end

bench "array iteration (for-in)"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  sum = 0
  for x in arr
    sum = sum + x
  end
end

bench "array index access (100 reads)"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  i = 0
  while i < 100
    x = arr[i % 10]
    i = i + 1
  end
end

bench "hash creation (5 pairs)"
  h = {"a" => 1, "b" => 2, "c" => 3, "d" => 4, "e" => 5}
end

bench "hash lookup (100 reads)"
  h = {"a" => 1, "b" => 2, "c" => 3, "d" => 4, "e" => 5}
  i = 0
  while i < 100
    x = h["c"]
    i = i + 1
  end
end

bench "array append (100 elements)"
  arr = []
  i = 0
  while i < 100
    arr = append(arr, i)
    i = i + 1
  end
end
