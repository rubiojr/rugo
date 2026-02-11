# Rugo Linter

A modular linter for Rugo source files, built in Rugo using the `ast` module. Each lint rule is a separate `.rugo` file with a standard interface.

## Usage

```
rugo run tools/linter/main.rugo -- <command> [--fix] <file(s)>
```

Or, after installing with `rugo tool install`:

```
rugo linter <command> [--fix] <file(s)>
```

### Commands

| Command | Description |
|---------|-------------|
| `smart-append` | Detect verbose append patterns |
| `all` | Run all registered linters |

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--fix` | `-f` | Auto-fix detected patterns in place |

## Examples

Scan a file for issues:

```
$ rugo run tools/linter/main.rugo -- smart-append examples/spawn.rugo
[smart-append] examples/spawn.rugo:35: tasks = append(tasks, t)
  suggestion: append tasks, t
1 issue(s) found.
```

Run all linters across multiple files:

```
$ rugo run tools/linter/main.rugo -- all examples/spawn.rugo examples/lambdas.rugo
[smart-append] examples/spawn.rugo:35: tasks = append(tasks, t)
  suggestion: append tasks, t
[smart-append] examples/lambdas.rugo:30: result = append(result, f(item))
  suggestion: append result, f(item)
2 issue(s) found.
```

Auto-fix in place:

```
$ rugo run tools/linter/main.rugo -- smart-append --fix examples/spawn.rugo
[smart-append] examples/spawn.rugo:35: fixed: tasks = append(tasks, t) → append tasks, t
1 issue(s) fixed.
```

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

## Architecture

```
tools/linter/
  linter.rugo              # CLI entry point — dispatches commands
  linters/
    smart_append.rugo      # smart-append lint rule
```

### Linter Registry

The main CLI maintains a `Linters` array where each entry provides a name and lambda-wrapped `lint`/`fix` functions:

```ruby
Linters = [
  {name: "smart-append", lint: fn(f) smart_append.lint(f) end, fix: fn(f, w) smart_append.fix(f, w) end}
]
```

The `all` command iterates over every entry. Individual commands dispatch to the matching linter.

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
    target: "arr",
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
| `fixed` | No | Suggested replacement (for display and `--fix`) |
| `target` | No | Variable/symbol name involved |

### `fix(path, warnings)`

Takes a file path and the warnings array from `lint()`. Rewrites the file in place. If the linter doesn't support auto-fix, implement as a no-op:

```ruby
def fix(path, warnings)
  # no-op for linters that can't auto-fix
end
```

### Registering

1. Add `require "./linters/your_linter"` to `linter.rugo`
2. Add an entry to the `Linters` array:

```ruby
Linters = [
  {name: "smart-append", lint: fn(f) smart_append.lint(f) end, fix: fn(f, w) smart_append.fix(f, w) end},
  {name: "your-rule", lint: fn(f) your_linter.lint(f) end, fix: fn(f, w) your_linter.fix(f, w) end}
]
```

That's it — the `all` command, `--fix` flag, output formatting, and file validation are handled by the framework.
