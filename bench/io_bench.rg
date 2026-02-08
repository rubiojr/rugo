# I/O and print benchmarks
# Run with: rugo bench bench/io_bench.rg 1>/dev/null
# Redirect stdout to /dev/null to avoid measuring terminal I/O
import "bench"

bench "puts single arg (100 calls)"
  i = 0
  while i < 100
    puts "hello"
    i = i + 1
  end
end

bench "puts multi arg (100 calls)"
  i = 0
  while i < 100
    puts "hello", "world", "foo"
    i = i + 1
  end
end
