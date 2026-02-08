# String operations benchmarks
# Tracks: string concatenation, interpolation, rugo_to_string overhead
use "bench"
use "conv"

bench "string concat (100 iterations)"
  s = ""
  i = 0
  while i < 100
    s = s + "x"
    i = i + 1
  end
end

bench "string interpolation"
  name = "world"
  age = 25
  s = "Hello #{name}, you are #{age} years old"
end

bench "conv.to_s on integers"
  i = 0
  while i < 100
    s = conv.to_s(i)
    i = i + 1
  end
end
