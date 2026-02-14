# Rugo RunMD

A documentation verifier for Rugo code blocks in markdown files. Parses markdown for `` ```ruby `` code blocks and validates each snippet compiles as valid Rugo. With `--verify`, it also runs snippets and compares output against an immediately following `` ```text `` block.

## Usage

```
rugo run tools/runmd/main.rugo -- <file.md> [file2.md ...]
```

Or, after installing with `rugo tool install`:

```
rugo runmd <file.md> [file2.md ...]
```

Multiple markdown files can be passed. Exit code 0 if all snippets compile (and verify), 1 if any fail.

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--interactive` | `-i` | Show each snippet and prompt before running |
| `--verify` | `-v` | Run snippets and verify output against following `` ```text `` blocks |

## Examples

Check all code blocks in a markdown file compile:

```
$ rugo runmd docs/quickstart.md
docs/quickstart.md:4: ok
docs/quickstart.md:10: ok

2 snippet(s), 2 passed, 0 failed
```

Detect a build failure:

```
$ rugo runmd broken.md
broken.md:4: FAIL
  expected identifier, got "end"

1 snippet(s), 0 passed, 1 failed
```

Verify documented output matches actual output:

```
$ rugo runmd --verify docs/quickstart.md
docs/quickstart.md:4: ok

1 snippet(s), 1 passed, 0 failed
```

When output mismatches, the expected and actual output are shown:

```
$ rugo runmd --verify bad_output.md
bad_output.md:4: FAIL
  expected: goodbye
  actual:   hello

1 snippet(s), 0 passed, 1 failed
```

Check multiple files at once:

```
$ rugo runmd docs/quickstart.md docs/language.md
```

### Interactive mode

With `-i`, each snippet is displayed (syntax-highlighted via `bat` when available) and you're prompted before running:

```
$ rugo runmd -i docs/quickstart.md

─── docs/quickstart.md:4 ───

puts "hello world"

Run? [y/N/q]
```

- **y** — compile and run the snippet
- **N** (default) — skip this snippet
- **q** — quit immediately

## How It Works

1. Parses the markdown file for fenced code blocks
2. Extracts all `` ```ruby `` blocks as Rugo snippets, recording their line numbers
3. For each snippet, writes it to a temp file and runs `rugo build` to check it compiles
4. In `--verify` mode, if a `` ```text `` block immediately follows a `` ```ruby `` block, the snippet is also executed and its output compared to the text block
5. Reports results per-snippet with `file:line` location
6. Cleans up temp files on exit

### Output verification convention

To document expected output for a snippet, place a `` ```text `` block directly after the `` ```ruby `` block:

````markdown
```ruby
puts "hello"
puts 42
```

```text
hello
42
```
````

Without `--verify`, the text block is ignored and only compilation is checked. With `--verify`, every `` ```ruby `` block **must** have a following `` ```text `` block — snippets without one are reported as failures. Output comparison normalizes line endings and trims trailing blank lines.

## Architecture

```
tools/runmd/
  main.rugo        # CLI entry point and core logic
```

Key functions:

| Function | Purpose |
|----------|---------|
| `extract_ruby_blocks(content)` | Parse markdown into snippet hashes with `code`, `line`, `expected`, `has_expected` |
| `normalize_output(raw)` | Normalize line endings and trim trailing blanks for stable comparison |
| `show_snippet(code)` | Display snippet with syntax highlighting (uses `bat` when available) |
| `draw_box(text, styler)` | Render Unicode box-drawing around text for interactive mode |
| `clean_output(raw)` | Strip temp file paths from build error messages |

Output is colorized in interactive mode. Set `NO_COLOR` to disable.
