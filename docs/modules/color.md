# color

ANSI terminal colors and styles. Respects the [NO_COLOR](https://no-color.org) convention.

```ruby
use "color"
```

## Foreground Colors

```ruby
puts color.red("error")
puts color.green("success")
puts color.yellow("warning")
puts color.blue("info")
puts color.magenta("special")
puts color.cyan("highlight")
puts color.white("bright")
puts color.gray("muted")
```

## Background Colors

```ruby
puts color.bg_red(" ERROR ")
puts color.bg_green(" OK ")
puts color.bg_yellow(" WARN ")
puts color.bg_blue(" INFO ")
puts color.bg_magenta(" SPECIAL ")
puts color.bg_cyan(" NOTE ")
puts color.bg_white(" LIGHT ")
puts color.bg_gray(" DIM ")
```

## Styles

```ruby
puts color.bold("important")
puts color.dim("secondary")
puts color.underline("linked")
```

## Composing

Colors and styles compose by nesting:

```ruby
puts color.bold(color.red("CRITICAL"))
puts color.white(color.bg_blue(" INFO "))
puts color.bold(color.underline("Title"))
```

## NO_COLOR

When the `NO_COLOR` environment variable is set, all functions return plain text without ANSI codes. This is automatic — no code changes needed.

## Example

```ruby
use "color"

puts color.bold("Deploy Status")
puts color.green("  ✓ API server")
puts color.green("  ✓ Database")
puts color.red("  ✗ Worker pool")
puts color.gray("  Last checked: 2m ago")
puts ""
puts color.white(color.bg_blue(" INFO ")) + " Deployment in progress"
```
