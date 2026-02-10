# Remote Modules

Rugo can load `.rg` modules directly from git repositories. This lets you
share and reuse Rugo code without any package registry.

## Basic Usage

```ruby
require "github.com/user/my-utils@v1.0.0" as "utils"
puts utils.slugify("Hello World")
```

The module is fetched via `git clone`, cached locally, and compiled into your
program. Tagged versions and commit SHAs are cached forever.

## Version Pinning

| Syntax | Meaning |
|--------|---------|
| `@v1.2.0` | Git tag (cached forever) |
| `@main` | Branch (re-fetched each build) |
| `@abc1234` | Commit SHA (cached forever) |
| *(none)* | Default branch (re-fetched) |

Always pin to a tag in production code:

```ruby
require "github.com/user/lib@v1.0.0" as "lib"
```

## Multi-File Libraries with `with`

The `with` clause selectively loads specific `.rg` files from a directory.
It works with both remote repositories and local directories.

### Local Directories

```ruby
# Given a directory mylib/ containing client.rg and helpers.rg:
require "mylib" with client, helpers

puts client.connect()
puts helpers.format("data")
```

The path must point to a directory (not a file). Each name in the `with`
list loads `<name>.rg` from that directory.

### Remote Repositories

```ruby
# Load only client and issue modules
require "github.com/rubiojr/rugh@v1.0.0" with client, issue

gh = client.from_env()
issues = issue.list(gh, "rubiojr", "rugo")
```

Each name in the `with` list loads `<name>.rg` from the repository root.
The filename becomes the namespace — `client.rg` becomes the `client`
namespace, `issue.rg` becomes `issue`, etc.

### How `with` Works

1. The repo is fetched once (same caching rules as regular requires)
2. Each named `.rg` file is loaded from the repo root
3. Each file gets its own namespace (the filename without `.rg`)
4. No entry point (`main.rg`) is needed

### Without `with`

Without `with`, Rugo looks for an entry point in this order:

1. `<repo-name>.rg` (e.g., `rugh.rg` for a repo named `rugh`)
2. `main.rg`
3. The sole `.rg` file (if there's exactly one)

If there are multiple `.rg` files and no entry point, you'll get an error
suggesting you use `with`.

## Publishing a Multi-File Module

Publishing a Rugo module is just pushing a git repo. No registry, no
manifest, no build step.

### Structure

```
my-lib/
  client.rg       # → client namespace
  helpers.rg      # → helpers namespace
  main.rg         # (optional) entry point for bare require
  README.md
```

### Rules for Module Authors

- Each `.rg` file at the repo root becomes a loadable module
- Function names are the public API — there's no `export` keyword
- Prefix private helpers with `_` by convention (e.g., `def _internal()`)
- If your library has inter-module dependencies (e.g., `helpers.rg` calls
  `client.get()`), document the required load order
- Add a `main.rg` if you want `require "..." ` (without `with`) to work

### Optional Entry Point

If you want users to load everything with a single require, add a `main.rg`:

```ruby
# main.rg — loads all sub-modules
require "./client"
require "./helpers"
```

Now both styles work:

```ruby
# Load everything
require "github.com/user/my-lib@v1.0.0"

# Or pick what you need
require "github.com/user/my-lib@v1.0.0" with client
```

## Inter-Module Dependencies

If one module calls functions from another (e.g., `issue.rg` calls
`client.get()`), the consumer must load both:

```ruby
require "github.com/user/lib@v1.0.0" with client, issue
# client is loaded first, so issue can call client.get()
```

Order matters — load dependencies before the modules that use them.

## Subpath Requires

You can also require a specific file by subpath:

```ruby
require "github.com/user/my-lib/client@v1.0.0"
# loads client.rg from the repo root, namespace "client"
```

This is an alternative to `with` when you only need a single module.

## Cache Location

Remote modules are cached in `~/.rugo/modules/` by default. Override with
the `RUGO_MODULE_DIR` environment variable:

```bash
RUGO_MODULE_DIR=/tmp/my-cache rugo run script.rg
```

## Lock File (`rugo.lock`)

When you build or run a script that uses remote modules, Rugo automatically
generates a `rugo.lock` file next to your source file. This records the exact
commit SHA for every remote dependency, making builds reproducible.

### Format

```
# rugo.lock — auto-generated, do not edit
github.com/user/repo v1.0.0 abc1234def5678901234567890abcdef12345678
github.com/user/utils _default 9f8e7d6c5b4a321098765432109876fedcba9876
```

Each line: `<module-path> <version-label> <resolved-sha>`

### How It Works

1. **First build**: Modules are fetched, SHAs recorded in `rugo.lock`
2. **Subsequent builds**: Locked SHAs are used — no network fetch needed
3. **Mutable versions** (`@main`, no version): Locked to their resolved SHA until explicitly updated
4. **Immutable versions** (`@v1.0.0`, `@abc1234`): Recorded for completeness and tamper detection

### Updating Dependencies

```bash
rugo update                    # re-resolve all mutable dependencies
rugo update github.com/user/repo  # re-resolve a specific module
```

Or delete `rugo.lock` and rebuild to re-resolve everything.

### Frozen Builds (CI)

Use `--frozen` with `rugo build` to fail if the lock file is missing or stale:

```bash
rugo build --frozen app.rg -o app
```

This ensures CI builds use exactly the locked versions. If a new dependency
is added but `rugo.lock` wasn't updated, the build fails immediately.

### Best Practices

- **Commit `rugo.lock`** to version control for reproducible builds
- Use `rugo update` to intentionally upgrade mutable dependencies
- Use `--frozen` in CI to catch unintentional dependency changes

---

**Quick reference:**

```ruby
# Single-file module
require "github.com/user/lib@v1.0.0" as "lib"

# Multi-file: load specific modules
require "github.com/user/lib@v1.0.0" with client, helpers

# Multi-file: load everything (needs main.rg in repo)
require "github.com/user/lib@v1.0.0"

# Subpath: single file from a multi-file repo
require "github.com/user/lib/client@v1.0.0"
```
