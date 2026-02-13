# Sandbox

Rugo supports opt-in process sandboxing using Linux [Landlock](https://landlock.io/). Add a `sandbox` directive to restrict what your script can access — filesystem paths and network ports.

## Basic Usage

```ruby
# Deny everything (maximum restriction)
sandbox

# Allow specific paths
sandbox ro: ["/etc"], rw: ["/tmp"], rox: ["/usr/bin"]

# Allow network
sandbox connect: [80, 443], bind: 8080
```

## Permission Types

| Keyword | Access | Example |
|---------|--------|---------|
| `ro` | Read-only | Config files |
| `rw` | Read + write | Temp/output dirs |
| `rox` | Read + execute | Binary dirs |
| `rwx` | Read + write + execute | Plugin dirs |
| `connect` | TCP connect | HTTP clients |
| `bind` | TCP bind | Servers |
| `env` | Env var allowlist | Restrict env access |

## CLI Flags

Apply sandbox restrictions without modifying the script:

```bash
rugo run --sandbox --ro /etc --rox /usr --connect 443 --env PATH script.rugo
```

CLI flags **override** any `sandbox` directive in the script.

## Important Notes

- **Linux only**: On other platforms, the directive is a no-op with a warning.
- **No auto-allows**: You must explicitly allow every path. Shell commands typically need `rox: ["/usr", "/lib"]` and `rw: ["/dev/null"]`.
- **Symlinks**: Landlock restricts the **target** path. Check with `readlink -f`.
- **stat() is not restricted**: `os.file_exists()` always works regardless of sandbox.

## Environment Variable Filtering

Restrict which env vars are visible (opt-in, works on all platforms):

```ruby
sandbox env: ["PATH", "HOME"]
import "os"
puts(os.getenv("HOME"))   # works
puts(os.getenv("SECRET"))  # empty string
```

## Shell Commands Example

```ruby
sandbox ro: ["/etc"], rox: ["/usr", "/lib"], rw: ["/dev/null"]
result = `cat /etc/resolv.conf`
puts(result)
```

For the full reference, see [docs/sandbox.md](../sandbox.md).
