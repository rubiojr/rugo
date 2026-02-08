# Control flow benchmarks
# Tracks: rugo_to_bool overhead, loop/branch efficiency
import "bench"

def noop()
  return nil
end

bench "while loop (10000 iterations)"
  i = 0
  while i < 10000
    i = i + 1
  end
end

bench "for-in with range array"
  arr = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10]
  sum = 0
  j = 0
  while j < 100
    for x in arr
      sum = sum + x
    end
    j = j + 1
  end
end

bench "if/elsif/else chain"
  i = 0
  while i < 1000
    if i < 250
      x = 1
    elsif i < 500
      x = 2
    elsif i < 750
      x = 3
    else
      x = 4
    end
    i = i + 1
  end
end

bench "function call overhead"
  i = 0
  while i < 1000
    noop()
    i = i + 1
  end
end
