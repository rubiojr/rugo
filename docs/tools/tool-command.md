# rugo tool

Manage Rugo CLI extensions. Tools are compiled Rugo programs installed to `~/.rugo/tools/` that extend the `rugo` CLI with custom subcommands — similar to how `git` discovers `git-*` binaries.

## Install

```
rugo tool install <path | remote-module | core>
```

### Local tool

Build and install a tool from a local directory:

```
$ rugo tool install ./tools/linter
installed linter → /home/user/.rugo/tools/rugo-linter
```

The directory must contain a `main.rugo` (or `<dirname>.rugo`) entry point.

### Remote tool

Install a tool from a remote git repository:

```
$ rugo tool install github.com/user/rugo-formatter@v1.0.0
installed rugo-formatter → /home/user/.rugo/tools/rugo-rugo-formatter
```

Uses the same module resolution as `require` — supports versions, branches, and SHAs.

### Core tools

Install all official tools from the Rugo repository:

```
$ rugo tool install core
installed linter → /home/user/.rugo/tools/rugo-linter
1 core tool(s) installed.
```

This fetches `github.com/rubiojr/rugo` and builds every subdirectory under `tools/`.

Pin to a specific version:

```
$ rugo tool install core@v0.50.0
```

## List

Show installed tools:

```
$ rugo tool list
linter               /home/user/.rugo/tools/rugo-linter
```

## Remove

Uninstall a tool:

```
$ rugo tool remove linter
removed linter
```

## Using installed tools

Once installed, tools are available as `rugo` subcommands:

```
$ rugo linter smart-append examples/spawn.rugo
[smart-append] examples/spawn.rugo:35: tasks = append(tasks, t)
  suggestion: append tasks, t
```

All arguments after the tool name are passed through to the tool binary.

## Writing a tool

A tool is any Rugo program in a directory with a `main.rugo` entry point:

```
my-tool/
  main.rugo        # entry point (required)
  linters/         # supporting modules (optional)
    foo.rugo
```

The tool name is derived from the directory name. Use `require` for internal modules and `use` for stdlib modules as usual.

### Description

Add a `# tool:` header comment on the first line of `main.rugo` to provide a description that appears in `rugo` help and `rugo tool list`:

```ruby
# tool: Format Rugo source files
```

This is extracted at install time and stored as a `.desc` sidecar file next to the binary.

### Conventions

- Entry point: `main.rugo`
- First line: `# tool: <one-line description>`
- Use the `cli` module for argument parsing
- Exit with `os.exit(1)` on errors
- Print to stdout for results, stderr for errors

## Environment

| Variable | Description |
|----------|-------------|
| `RUGO_TOOLS_DIR` | Override the tools directory (default: `~/.rugo/tools`). Useful for testing. |
