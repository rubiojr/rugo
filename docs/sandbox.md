# Sandbox (Landlock)

Rugo supports opt-in process sandboxing using Linux [Landlock](https://landlock.io/), a kernel-native security module that lets unprivileged processes restrict themselves. When a Rugo script declares a `sandbox` directive, the compiled binary self-restricts before executing any user code — no root, no containers, no SELinux/AppArmor configs needed.

## How It Works

Landlock is a **self-sandboxing** mechanism: the process applies restrictions to itself, and these restrictions are inherited by all child processes (including shell commands). Once applied, restrictions **cannot be relaxed** — only further restricted.

Rugo leverages [go-landlock](https://github.com/landlock-lsm/go-landlock) to inject Landlock syscalls into the compiled binary. The sandboxing code runs in `main()` immediately after the panic handler and **before any user code executes**.

### Compilation Flow

```
.rugo source (with sandbox directive)
   │
   ▼
Parser recognizes SandboxStmt
   │
   ▼
Codegen emits go-landlock calls in main()
   │
   ▼
Generated go.mod includes go-landlock dependency
   │
   ▼
Compiled binary self-restricts on startup
```

## Requirements

| Feature | Minimum Kernel | Landlock ABI |
|---------|---------------|--------------|
| Filesystem sandboxing | Linux 5.13 | v1 |
| File referring/reparenting | Linux 5.19 | v2 |
| File truncation control | Linux 6.2 | v3 |
| TCP network restrictions | Linux 6.7 | v4 |
| IOCTL on special files | Linux 6.10 | v5 |

Rugo uses `landlock.V5.BestEffort()` which gracefully degrades on older kernels — the highest available ABI version is used automatically.

On **non-Linux** systems, the sandbox directive is a **no-op** that prints a warning to stderr:

```
rugo: warning: sandbox requires Linux with Landlock support, running unrestricted
```

## Syntax

### Script Directive

The `sandbox` directive is a top-level statement. Place it at the beginning of your script, before any code that performs I/O.

```ruby
# Bare sandbox: deny ALL filesystem and network access
sandbox

# With permissions (single values)
sandbox ro: "/etc", rw: "/tmp"

# With permissions (arrays)
sandbox ro: ["/etc", "/usr/share"], rw: ["/tmp", "/var/log"]

# Full example with all permission types
sandbox ro: ["/etc"], rw: ["/tmp"], rox: ["/usr/bin"], rwx: ["/var/data"], connect: [80, 443], bind: 8080
```

### Permission Types

| Keyword | Access | Use Case |
|---------|--------|----------|
| `ro` | Read-only | Config files, data files |
| `rw` | Read + write | Temp dirs, log files, output dirs |
| `rox` | Read + execute | Binary directories (`/usr/bin`, `/usr/lib`) |
| `rwx` | Read + write + execute | Plugin dirs, build dirs |
| `connect` | TCP connect to port | HTTP clients, database connections |
| `bind` | TCP bind to port | Servers, listeners |

### Filesystem Permissions Detail

Each permission type maps to specific Landlock access rights:

**`ro` (read-only)**
- Files: `ReadFile`
- Directories: `ReadFile`, `ReadDir`

**`rw` (read-write)**
- Files: `ReadFile`, `WriteFile`, `Truncate`, `IoctlDev`
- Directories: all of the above plus `ReadDir`, `RemoveDir`, `RemoveFile`, `MakeChar`, `MakeDir`, `MakeReg`, `MakeSock`, `MakeFifo`, `MakeBlock`, `MakeSym`, `Refer`

**`rox` (read + execute)**
- Files: `Execute`, `ReadFile`
- Directories: `Execute`, `ReadFile`, `ReadDir`

**`rwx` (read + write + execute)**
- All `rw` rights plus `Execute`

### CLI Flags

You can apply sandbox restrictions from the command line without modifying the script:

```bash
# Bare sandbox (deny everything)
rugo run --sandbox script.rugo

# With permissions (repeatable flags)
rugo run --sandbox --ro /etc --ro /usr/share --rw /tmp --connect 443 script.rugo

# Build a sandboxed binary
rugo build --sandbox --ro /etc --rox /usr -o mybinary script.rugo
```

Available flags (all repeatable):

| Flag | Description |
|------|-------------|
| `--sandbox` | Enable sandboxing (required to activate) |
| `--ro PATH` | Allow read-only access |
| `--rw PATH` | Allow read-write access |
| `--rox PATH` | Allow read + execute access |
| `--rwx PATH` | Allow read + write + execute access |
| `--connect PORT` | Allow TCP connections to port |
| `--bind PORT` | Allow TCP bind to port |

### CLI Override

When both a script directive and CLI flags are present, **CLI flags override the script directive entirely**. The script's `sandbox` permissions are ignored.

```ruby
# script.rugo
sandbox ro: ["/etc", "/tmp"]   # ← ignored when CLI flags are used
puts("hello")
```

```bash
# CLI overrides: only /etc is allowed, /tmp is denied
rugo run --sandbox --ro /etc script.rugo
```

## Important Notes

### What Landlock Restricts

Landlock restricts **content access** operations:
- `open()`, `read()`, `write()`, `execute()`, `truncate()`
- Directory operations: create/remove files, create directories, symlinks
- Network: TCP `bind()` and `connect()`

### What Landlock Does NOT Restrict

- **`stat()` / `lstat()`**: File metadata queries always succeed. `os.file_exists()` returns `true` even for paths outside the sandbox.
- **Existing file descriptors**: Files opened before sandboxing are not affected.
- **UDP and other protocols**: Only TCP is restricted (as of Landlock ABI v5).
- **Process operations**: fork, exec (of allowed binaries), signals, etc.

### Symlinks

Landlock restricts access to the **target** of a symlink, not the symlink itself. If `/etc/os-release` is a symlink to `/usr/lib/os-release`, you need to allow `/usr/lib` (or the specific file), not just `/etc`.

### Shell Commands

Shell commands (backticks, `os.exec()`) run as child processes that inherit sandbox restrictions. For shell commands to work, you typically need:

```ruby
sandbox rox: ["/usr", "/lib"], rw: ["/dev/null"], ro: ["/etc"]
```

- `/usr` and `/lib`: executable binaries and shared libraries
- `/dev/null`: required by shell I/O redirection
- `/etc`: often needed for DNS resolution (`/etc/resolv.conf`), but note symlinks

### No Auto-Allows

Rugo does **not** automatically add any paths. If your script uses shell commands, you must explicitly allow every path needed. This is by design — the sandbox is only useful if you understand and control what it allows.

### Best-Effort Mode

Rugo uses Landlock's best-effort mode by default. This means:
- On **Linux 6.10+**: Full filesystem + network + IOCTL restrictions (ABI v5)
- On **Linux 6.7-6.9**: Filesystem + network, no IOCTL control (ABI v4)
- On **Linux 5.13-6.6**: Filesystem only, no network restrictions
- On **older kernels**: Sandbox is a no-op (warning printed)

If Landlock fails to apply, a warning is printed to stderr but the program continues running unrestricted.

## Examples

### Read-only script that processes config files

```ruby
sandbox ro: ["/etc"]
use "os"

# Can stat /etc files
if os.file_exists("/etc/hosts")
  puts("hosts file exists")
end

# Cannot write, execute, or access network
```

### HTTP client with network restrictions

```ruby
sandbox ro: ["/etc"], rox: ["/usr", "/lib"], rw: ["/dev/null"], connect: [443]
use "http"

# Can only connect to port 443 (HTTPS)
response = http.get("https://example.com")
puts(response["body"])
```

### Web server with minimal permissions

```ruby
sandbox ro: ["/etc", "/var/www"], rox: ["/usr", "/lib"], bind: [8080], connect: [5432]
use "web"

# Bind to port 8080, connect to PostgreSQL on 5432
# Read-only access to static files
web.get("/", "index")

def index(req)
  return {status: 200, body: "hello"}
end

web.run(8080)
```

## Troubleshooting

### "Permission denied" errors

1. **Use `rugo emit` to inspect generated code**: `rugo emit script.rugo | grep rugo_sandbox` shows the exact Landlock rules being applied.
2. **Check symlinks**: `readlink -f /path/to/file` to find the real target path.
3. **Add paths incrementally**: Start with a broad sandbox and narrow it down.
4. **Shell commands**: Remember `/usr`, `/lib`, and `/dev/null` are usually needed.

### "sandbox requires Linux" warning

The script is running on a non-Linux system. The sandbox directive is silently skipped and the script runs unrestricted.

### Check Landlock availability

```bash
# Check if Landlock is enabled in your kernel
cat /sys/kernel/security/lsm
# Should include "landlock" in the list

# Check kernel version (need 5.13+ for basic, 6.7+ for network)
uname -r
```

### Landlock not enforcing

If restrictions don't seem to apply:
1. Verify Landlock is in the LSM list: `cat /sys/kernel/security/lsm`
2. Check that you're on a supported kernel version
3. Some container runtimes may disable Landlock — check your container security profile

## Implementation Details

The sandbox implementation touches these parts of the Rugo codebase:

- **Grammar**: `parser/rugo.ebnf` — `SandboxStmt`, `SandboxPerm`, `SandboxList` rules
- **AST**: `ast/nodes.go` — `SandboxStmt` node with `RO`, `RW`, `ROX`, `RWX`, `Connect`, `Bind` fields
- **Walker**: `ast/walker.go` — `walkSandboxStmt()` and permission parsing helpers
- **Preprocessor**: `ast/preprocess.go` — `sandbox` keyword registration, colon syntax exemption, semicolon separator injection
- **Codegen**: `compiler/codegen.go` — `writeSandboxRuntime()` (helper functions) and `writeSandboxApply()` (main() injection)
- **Compiler**: `compiler/compiler.go` — `SandboxConfig` type, go-landlock dependency injection in `goModContent()`
- **CLI**: `cmd/cmd.go` — `parseSandboxFlags()` for `--sandbox` flag extraction

### Generated Code Structure

When a sandbox directive is present, the codegen produces:

```go
import (
    "runtime"
    "github.com/landlock-lsm/go-landlock/landlock"
    llsyscall "github.com/landlock-lsm/go-landlock/landlock/syscall"
)

// Helper functions for access right construction
func rugo_sandbox_fs_ro(dir bool) landlock.AccessFSSet { ... }
func rugo_sandbox_fs_rw(dir bool) landlock.AccessFSSet { ... }
func rugo_sandbox_fs_rox(dir bool) landlock.AccessFSSet { ... }
func rugo_sandbox_fs_rwx(dir bool) landlock.AccessFSSet { ... }
func rugo_sandbox_is_dir(path string) bool { ... }

func main() {
    defer func() { /* panic handler */ }()

    if runtime.GOOS != "linux" {
        // warn and continue unrestricted
    } else {
        cfg := landlock.V5.BestEffort()
        // Build and apply rules...
        cfg.RestrictPaths(fsRules...)
        cfg.RestrictNet(netRules...)
    }

    // User code runs here, fully sandboxed
}
```

### Dependencies

The sandbox feature adds two Go module dependencies to the generated program:

- `github.com/landlock-lsm/go-landlock` — Go bindings for Landlock LSM
- `kernel.org/pub/linux/libs/security/libcap/psx` — Required by go-landlock for thread-safe syscalls

These are only added to the generated `go.mod` when a `sandbox` directive is present (either in the script or via CLI flags).
