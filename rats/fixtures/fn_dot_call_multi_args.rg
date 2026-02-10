math = {
  clamp: fn(val, lo, hi)
    if val < lo
      return lo
    end
    if val > hi
      return hi
    end
    return val
  end
}
puts math.clamp(5, 0, 10)
puts math.clamp(-1, 0, 10)
puts math.clamp(99, 0, 10)
