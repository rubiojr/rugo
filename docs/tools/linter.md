# Rugo Linter

A modular linter for Rugo source files, built in Rugo using the `ast` module. Each lint rule is a separate `.rugo` file with a standard interface.

## Usage

```
rugo run tools/linter/main.rugo -- <command> [--fix] <file(s)/dir(s)>
```

Or, after installing with `rugo tool install`:

```
rugo linter <command> [--fix] <file(s)/dir(s)>
```

Arguments can be individual `.rugo` files or directories. When a directory is given, all `.rugo` files are scanned recursively.

### Commands

| Command | Description |
|---------|-------------|
| `smart-append` | Detect verbose append patterns |
| `string-interp` | Detect string concatenation over interpolation |
| `all` | Run all registered linters |

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--fix` | `-f` | Auto-fix detected patterns in place |

## Examples

Scan a file for issues:

```
$ rugo linter smart-append examples/spawn.rugo
[smart-append] examples/spawn.rugo:35: tasks = append(tasks, t)
  suggestion: append tasks, t
1 issue(s) found.
```

Scan a directory recursively:

```
$ rugo linter all src/
[string-interp] src/client.rugo:64: msg = "API error " + conv.to_s(code)
  suggestion: msg = "API error #{code}"
1 issue(s) found.
```

Auto-fix in place:

```
$ rugo linter string-interp --fix examples/variables.rugo
[string-interp] examples/variables.rugo:3: fixed: greeting = "Hello, " + name + "!" → greeting = "Hello, #{name}!"
1 issue(s) fixed.
```

Output is colorized when running in an interactive terminal. Set `NO_COLOR` to disable.

## Available Linters

### smart-append

Detects `x = append(x, val)` patterns that can be replaced with the bare append sugar `append x, val`.

**What it catches:**

```ruby
# Before (verbose)
arr = append(arr, 1)
results = append(results, item)

# After (idiomatic)
append arr, 1
append results, item
```

**Fix mode:** Rewrites the line in place. Always safe — the bare append desugars to the same code.

**Skips:**
- Cross-variable assignments: `other = append(arr, 1)` (different semantics)
- Non-identifier first args: `append(42, 5)` (runtime error, not a rewrite candidate)
- Append used as an expression: `puts(append(arr, 1))`

### string-interp

Detects string concatenation patterns that should use interpolation instead.

**What it catches:**

```ruby
# Before (verbose)
greeting = "Hello, " + name + "!"
puts("Count: " + conv.to_s(n))
msg = conv.to_s(count) + " items"

# After (idiomatic)
greeting = "Hello, #{name}!"
puts("Count: #{n}")
msg = "#{count} items"
```

**Fix mode:** Rewrites the concatenation chain into a single interpolated string. Automatically unwraps `conv.to_s()` wrappers since `#{}` calls `__to_s()` internally.

**Skips:**
- Single-quoted (raw) strings — never flag
- Comments — lines starting with `#`
- Expressions containing double quotes inside `#{}` (e.g., `h["key"]`) — warns but does not auto-fix
- Non-string concatenation (`a + b` where neither is a string literal)
- Multi-line concatenation (V1 limitation)

## Architecture

```
tools/linter/
  main.rugo                # CLI entry point — dispatches commands
  linters/
    smart_append.rugo      # smart-append lint rule
    string_interp.rugo     # string-interp lint rule
```

### Linter Registry

The main CLI maintains a `Linters` array where each entry provides a name and lambda-wrapped `lint`/`fix` functions:

```ruby
Linters = [
  {name: "smart-append", lint: fn(f) smart_append.lint(f) end, fix: fn(f, w) smart_append.fix(f, w) end},
  {name: "string-interp", lint: fn(f) string_interp.lint(f) end, fix: fn(f, w) string_interp.fix(f, w) end}
]
```

The `all` command iterates over every entry. Individual commands dispatch to the matching linter.

### Directory Scanning

When a directory is passed as an argument, the linter recursively finds all `.rugo` files and expands them into a single file list. This expansion happens once, before any linters run, so `all` doesn't re-scan per linter.

## Writing a New Linter

Each linter is a `.rugo` file in `linters/` that exposes two functions:

### `lint(path)`

Takes a file path, returns an array of warning hashes:

```ruby
def lint(path)
  warnings = []
  prog = ast.parse_file(path)
  # ... walk AST, detect issues ...
  # Each warning is a hash:
  append warnings, {
    path: path,
    line: 10,
    source: "arr = append(arr, 1)",
    fixed: "append arr, 1"
  }
  return warnings
end
```

Warning hash keys:

| Key | Required | Description |
|-----|----------|-------------|
| `path` | Yes | File path |
| `line` | Yes | Line number (1-indexed) |
| `source` | Yes | Original source line (trimmed) |
| `fixed` | Yes | Suggested replacement (for display and `--fix`) |

### `fix(path, warnings)`

Takes a file path and the warnings array from `lint()`. Rewrites the file in place. If the linter doesn't support auto-fix, implement as a no-op:

```ruby
def fix(path, warnings)
  # no-op for linters that can't auto-fix
end
```

### Registering

1. Add `require "./linters/your_linter"` to `main.rugo`
2. Add an entry to the `Linters` array:

```ruby
Linters = [
  # ... existing linters ...
  {name: "your-rule", lint: fn(f) your_linter.lint(f) end, fix: fn(f, w) your_linter.fix(f, w) end}
]
```

That's it — the `all` command, `--fix` flag, output formatting, directory scanning, and file validation are handled by the framework.
