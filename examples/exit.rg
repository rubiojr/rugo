# exit - terminate the program with an optional exit code
#
# exit with no args defaults to code 0 (success)
# exit(code) exits with the given integer code

puts "doing work..."

ok = true

if !ok
  puts "something went wrong"
  exit 1
end

puts "all good!"
exit
