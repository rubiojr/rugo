# 20. Doc Comments

Rugo uses `#` comments for documentation. Write doc comments immediately before
`def` or `struct` declarations with no blank line gap.

## Convention

```ruby
# Calculates the factorial of n.
# Returns 1 when n <= 1.
def factorial(n)
  if n <= 1
    return 1
  end
  return n * factorial(n - 1)
end
```

The rules are simple:

- **Doc comment**: consecutive `#` lines immediately before `def`/`struct` (no blank line gap)
- **File-level doc**: first `#` block at top of file, before any code
- **Regular comment**: `#` inside function bodies, or after a blank line gap — never shown by `rugo doc`

## `rugo doc` Command

```bash
# Show docs for a .rugo file
rugo doc myfile.rugo

# Show docs for a specific function or struct
rugo doc myfile.rugo factorial

# Show docs for a stdlib module
rugo doc http

# Show docs for a Go bridge package
rugo doc strings
rugo doc time

# Disambiguate when a name exists as both module and bridge (e.g. os, json)
rugo doc use:os            # force stdlib module
rugo doc import:os         # force bridge package

# Show docs for a remote module
rugo doc github.com/user/repo

# List all available modules and packages
rugo doc --all
```

When `bat` is installed, output is syntax-highlighted automatically.
Set `NO_COLOR=1` to disable.

## Example

Create a file `lib.rugo`:

```ruby
# Math utility library.

# Adds two numbers together.
def add(a, b)
  return a + b
end

# A 2D point.
struct Point
  x
  y
end
```

Then run:

```
$ rugo doc lib.rugo
Math utility library.

struct Point { x, y }
    A 2D point.

def add(a, b)
    Adds two numbers together.
```

---

Next: check out the [Examples](../../examples/) directory.
