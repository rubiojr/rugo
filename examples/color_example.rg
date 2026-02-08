use "color"

puts color.bold("Deploy Status")
puts color.green("  ✓ API server")
puts color.green("  ✓ Database")
puts color.red("  ✗ Worker pool")
puts color.yellow("  ⚠ Cache warming")
puts color.gray("  Last checked: 2m ago")
puts ""
puts color.white(color.bg_blue(" INFO ")) + " Deployment in progress"
puts color.white(color.bg_red(" ERROR ")) + " Worker pool unreachable"
puts color.bold(color.underline("Done."))
