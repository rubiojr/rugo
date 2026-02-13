package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rubiojr/rugo/cmd/dev"
	"github.com/rubiojr/rugo/compiler"
	rugodoc "github.com/rubiojr/rugo/doc"
	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/remote"
	"github.com/urfave/cli/v3"
	"golang.org/x/term"
)

// Execute runs the Rugo CLI with the given version string.
// Import modules via blank imports before calling this function
// so they register via init().
func Execute(version string) {
	cmd := &cli.Command{
		Name:                   "rugo",
		Usage:                  "A Ruby-inspired language that compiles to Go",
		Version:                version,
		UseShortOptionHandling: true,
		// Allow `rugo script.rugo` as shorthand for `rugo run script.rugo`
		// Also dispatch to installed tools: `rugo linter` → `~/.rugo/tools/rugo-linter`
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() > 0 {
				arg := cmd.Args().First()
				if compiler.IsRugoFile(arg) || isRugoScript(arg) {
					comp := &compiler.Compiler{}
					return comp.Run(arg, cmd.Args().Tail()...)
				}
				// Check if it's an installed tool
				if err := execTool(arg, cmd.Args().Tail()); err == nil {
					return nil // execTool replaces the process
				}
			}
			return cli.DefaultShowRootCommandHelp(cmd)
		},
		Commands: []*cli.Command{
			{
				Name:            "run",
				Usage:           "Compile and run a Rugo source file",
				ArgsUsage:       "<file.rugo> [args...]",
				SkipFlagParsing: true,
				Action:          runAction,
			},
			{
				Name:      "build",
				Usage:     "Compile a Rugo source file to a native binary",
				ArgsUsage: "<file.rugo>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Output binary name",
					},
					&cli.BoolFlag{
						Name:  "frozen",
						Usage: "Error if rugo.lock is stale or a new dependency needs resolving",
					},
				},
				Action: buildAction,
			},
			{
				Name:      "emit",
				Usage:     "Output the generated Go source code",
				ArgsUsage: "<file.rugo>",
				Action:    emitAction,
			},
			{
				Name:            "eval",
				Usage:           "Evaluate Rugo source from an argument or stdin",
				ArgsUsage:       "[source]",
				SkipFlagParsing: true,
				Action:          evalAction,
			},
			{
				Name:      "rats",
				Usage:     "Run tests from _test.rugo files and Rugo files with inline rats blocks",
				ArgsUsage: "[file.rugo | directory]",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "filter",
						Aliases: []string{"f"},
						Usage:   "Run only tests matching this substring",
					},
					&cli.IntFlag{
						Name:    "jobs",
						Aliases: []string{"j"},
						Usage:   "Parallel test files",
						Value:   4,
					},
					&cli.BoolFlag{
						Name:    "no-color",
						Aliases: []string{"C"},
						Usage:   "Disable ANSI color output",
					},
					&cli.IntFlag{
						Name:    "timeout",
						Aliases: []string{"t"},
						Usage:   "Per-test timeout in seconds (0 to disable)",
						Value:   30,
					},
					&cli.BoolFlag{
						Name:  "timing",
						Usage: "Show per-test and total elapsed time",
					},
					&cli.BoolFlag{
						Name:  "recap",
						Usage: "Print all failures with details at the end",
					},
				},
				Action: testAction,
			},
			{
				Name:      "bench",
				Usage:     "Run _bench.rugo benchmark files",
				ArgsUsage: "[file_bench.rugo | directory]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "no-color",
						Aliases: []string{"C"},
						Usage:   "Disable ANSI color output",
					},
				},
				Action: benchAction,
			},
			dev.Command(),
			{
				Name:      "doc",
				Usage:     "Show documentation for files, modules, and bridge packages",
				ArgsUsage: "<file.rugo|[use:|import:]module|package> [symbol]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "all",
						Aliases: []string{"a"},
						Usage:   "List all available modules and bridge packages",
					},
				},
				Action: docAction,
			},
			{
				Name:  "mod",
				Usage: "Manage remote module dependencies",
				Commands: []*cli.Command{
					{
						Name:      "tidy",
						Usage:     "Resolve remote dependencies and write rugo.lock",
						ArgsUsage: "",
						Action:    modTidyAction,
					},
					{
						Name:      "update",
						Usage:     "Re-resolve mutable remote dependencies and update rugo.lock",
						ArgsUsage: "[module-path]",
						Action:    modUpdateAction,
					},
				},
			},
			{
				Name:  "tool",
				Usage: "Manage Rugo CLI tool extensions",
				Commands: []*cli.Command{
					{
						Name:            "install",
						Usage:           "Build and install a tool from a local path, remote module, or 'core'",
						ArgsUsage:       "<path | remote-module | core>",
						SkipFlagParsing: true,
						Action:          toolInstallAction,
					},
					{
						Name:   "list",
						Usage:  "List installed tools",
						Action: toolListAction,
					},
					{
						Name:      "remove",
						Usage:     "Remove an installed tool",
						ArgsUsage: "<name>",
						Action:    toolRemoveAction,
					},
				},
			},
		},
	}

	// Inject installed tools as top-level commands so they appear in help.
	for _, tc := range installedToolCommands() {
		cmd.Commands = append(cmd.Commands, tc)
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, formatError(err.Error()))
		os.Exit(1)
	}
}

func runAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo run <file.rugo> [args...]")
	}
	args := cmd.Args().Slice()
	sandbox, args := parseSandboxFlags(args)
	if len(args) == 0 {
		return fmt.Errorf("usage: rugo run [--sandbox flags...] <file.rugo> [args...]")
	}
	comp := &compiler.Compiler{Sandbox: sandbox}
	return comp.Run(args[0], args[1:]...)
}

func buildAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo build [-o output] [--frozen] [--sandbox flags...] <file.rugo>")
	}
	sandbox, _ := parseSandboxFlags(cmd.Args().Slice())
	comp := &compiler.Compiler{Frozen: cmd.Bool("frozen"), Sandbox: sandbox}
	output := cmd.String("output")
	// Also check if -o was passed after the filename (urfave quirk)
	if output == "" {
		for i, arg := range os.Args {
			if (arg == "-o" || arg == "--output") && i+1 < len(os.Args) {
				output = os.Args[i+1]
			}
		}
	}
	return comp.Build(cmd.Args().First(), output)
}

// parseSandboxFlags extracts --sandbox and related flags from args.
// Returns the SandboxConfig (nil if --sandbox not present) and remaining args.
func parseSandboxFlags(args []string) (*compiler.SandboxConfig, []string) {
	hasSandbox := false
	var ro, rw, rox, rwx []string
	var connect, bind []int
	var env []string
	var remaining []string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--sandbox":
			hasSandbox = true
		case "--ro":
			if i+1 < len(args) {
				i++
				ro = append(ro, args[i])
			}
		case "--rw":
			if i+1 < len(args) {
				i++
				rw = append(rw, args[i])
			}
		case "--rox":
			if i+1 < len(args) {
				i++
				rox = append(rox, args[i])
			}
		case "--rwx":
			if i+1 < len(args) {
				i++
				rwx = append(rwx, args[i])
			}
		case "--connect":
			if i+1 < len(args) {
				i++
				if p, err := strconv.Atoi(args[i]); err == nil {
					connect = append(connect, p)
				}
			}
		case "--bind":
			if i+1 < len(args) {
				i++
				if p, err := strconv.Atoi(args[i]); err == nil {
					bind = append(bind, p)
				}
			}
		case "--env":
			if i+1 < len(args) {
				i++
				env = append(env, args[i])
			}
		default:
			remaining = append(remaining, args[i])
		}
	}

	if !hasSandbox {
		return nil, args
	}
	return &compiler.SandboxConfig{
		RO: ro, RW: rw, ROX: rox, RWX: rwx,
		Connect: connect, Bind: bind, Env: env, EnvSet: len(env) > 0,
	}, remaining
}

func emitAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo emit <file.rugo>")
	}
	comp := &compiler.Compiler{}
	src, err := comp.Emit(cmd.Args().First())
	if err != nil {
		return err
	}
	fmt.Print(src)
	return nil
}

func evalAction(ctx context.Context, cmd *cli.Command) error {
	var source string
	if cmd.NArg() > 0 {
		source = strings.Join(cmd.Args().Slice(), " ")
	} else if !term.IsTerminal(int(os.Stdin.Fd())) {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		source = string(data)
	}

	if strings.TrimSpace(source) == "" {
		return fmt.Errorf("usage: rugo eval '<source>' or echo '<source>' | rugo eval")
	}

	tmpDir, err := os.MkdirTemp("", "rugo-eval-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	srcFile := filepath.Join(tmpDir, "eval.rugo")
	if err := os.WriteFile(srcFile, []byte(source), 0644); err != nil {
		return fmt.Errorf("writing source: %w", err)
	}

	comp := &compiler.Compiler{BaseDir: tmpDir}
	return comp.Run(srcFile)
}

func modTidyAction(ctx context.Context, cmd *cli.Command) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	// Find all .rugo files in the current directory (non-recursive).
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading directory: %w", err)
	}
	var rugoFiles []string
	for _, e := range entries {
		if !e.IsDir() && compiler.IsRugoFile(e.Name()) {
			rugoFiles = append(rugoFiles, filepath.Join(dir, e.Name()))
		}
	}
	if len(rugoFiles) == 0 {
		fmt.Fprintln(os.Stderr, "no .rugo files found in current directory")
		return nil
	}

	// Create a shared resolver for all files.
	r := &remote.Resolver{SuppressHint: true}
	if err := r.InitLockFromDir(dir); err != nil {
		return err
	}

	// Track which file introduced each module version for conflict detection.
	type modSource struct {
		version string
		file    string
	}
	moduleVersions := make(map[string]modSource) // module path → first source

	// Compile each file to discover and resolve remote dependencies.
	for _, f := range rugoFiles {
		keysBefore := copyStringSet(r.ResolvedKeys())

		comp := &compiler.Compiler{Resolver: r}
		// Compile triggers require resolution; we discard the output.
		if _, err := comp.Compile(f); err != nil {
			// Skip files that fail to compile (syntax errors, etc.)
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %s\n", filepath.Base(f), err)
			continue
		}

		// Check new resolved keys for version conflicts.
		for key := range r.ResolvedKeys() {
			if keysBefore[key] {
				continue
			}
			mod, ver := remote.SplitLockKey(key)
			if mod == "" {
				continue
			}
			if existing, ok := moduleVersions[mod]; ok && existing.version != ver {
				return fmt.Errorf("version conflict for %s:\n  %s requires %s\n  %s requires %s\nAlign on a single version, then re-run 'rugo mod tidy'.",
					mod, existing.file, existing.version, filepath.Base(f), ver)
			}
			moduleVersions[mod] = modSource{version: ver, file: filepath.Base(f)}
		}
	}

	// Prune entries that were not resolved during this tidy run.
	if keys := r.ResolvedKeys(); keys != nil {
		r.LockFile().Prune(keys)
	} else {
		// No remote deps resolved — remove the lock file if it exists.
		r.LockFile().Prune(nil)
	}

	return r.WriteLock()
}

// copyStringSet returns a shallow copy of a string set.
func copyStringSet(m map[string]bool) map[string]bool {
	if m == nil {
		return nil
	}
	c := make(map[string]bool, len(m))
	for k := range m {
		c[k] = true
	}
	return c
}

func modUpdateAction(ctx context.Context, cmd *cli.Command) error {
	// Find the rugo.lock in the current directory.
	lockPath, err := filepath.Abs("rugo.lock")
	if err != nil {
		return err
	}

	lf, err := remote.ReadLockFile(lockPath)
	if err != nil {
		return err
	}
	if len(lf.Entries) == 0 {
		fmt.Fprintln(os.Stderr, "no rugo.lock found or lock file is empty")
		return nil
	}

	r := &remote.Resolver{}
	r.InitLock(lockPath, lf)

	module := ""
	if cmd.NArg() > 0 {
		module = cmd.Args().First()
	}

	return r.UpdateEntry(module)
}

// isRugoScript checks if a file exists and looks like a rugo script.
func isRugoScript(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	line := string(buf[:n])
	return strings.HasPrefix(line, "#!")
}

func docAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.Bool("all") {
		docOutput(rugodoc.FormatAllModules())
		return nil
	}

	if cmd.NArg() < 1 {
		docOutput(rugodoc.FormatAllModules())
		return nil
	}

	target := cmd.Args().First()
	symbol := ""
	if cmd.NArg() > 1 {
		symbol = cmd.Args().Get(1)
	}

	// Parse use:/import: prefix for disambiguation
	forceModule, forceBridge := false, false
	if strings.HasPrefix(target, "use:") {
		forceModule = true
		target = strings.TrimPrefix(target, "use:")
	} else if strings.HasPrefix(target, "import:") {
		forceBridge = true
		target = strings.TrimPrefix(target, "import:")
	}

	// Forced module lookup (use:)
	if forceModule {
		m, ok := modules.Get(target)
		if !ok {
			return fmt.Errorf("module %q not found", target)
		}
		docOutput(rugodoc.FormatModule(m))
		return nil
	}

	// Forced bridge lookup (import:)
	if forceBridge {
		if pkg, ok := gobridge.LookupByNS(target); ok {
			docOutput(rugodoc.FormatBridgePackage(pkg))
			return nil
		}
		if pkg := gobridge.GetPackage(target); pkg != nil {
			docOutput(rugodoc.FormatBridgePackage(pkg))
			return nil
		}
		return fmt.Errorf("bridge package %q not found", target)
	}

	// Mode 1: Rugo source file
	if compiler.IsRugoFile(target) {
		fd, err := rugodoc.ExtractFile(target)
		if err != nil {
			return fmt.Errorf("reading %s: %w", target, err)
		}
		if symbol != "" {
			doc, sig, found := rugodoc.LookupSymbol(fd, symbol)
			if !found {
				return fmt.Errorf("%s: symbol %q not found", target, symbol)
			}
			docOutput(rugodoc.FormatSymbol(doc, sig))
			return nil
		}
		docOutput(rugodoc.FormatFile(fd))
		return nil
	}

	// Mode 2: local directory (e.g. ./gummy, gummy, .)
	if info, err := os.Stat(target); err == nil && info.IsDir() {
		return docLocalDir(target, symbol)
	}

	// Check for ambiguity: name exists as both module and bridge package
	_, isModule := modules.Get(target)
	_, isBridge := gobridge.LookupByNS(target)
	if isModule && isBridge {
		return fmt.Errorf("%q is both a module and a bridge package, use \"rugo doc use:%s\" or \"rugo doc import:%s\"", target, target, target)
	}

	// Mode 3: stdlib module (use)
	if isModule {
		m, _ := modules.Get(target)
		docOutput(rugodoc.FormatModule(m))
		return nil
	}

	// Mode 4: bridge package by namespace (import)
	if isBridge {
		pkg, _ := gobridge.LookupByNS(target)
		docOutput(rugodoc.FormatBridgePackage(pkg))
		return nil
	}

	// Mode 5: bridge package by full path
	if pkg := gobridge.GetPackage(target); pkg != nil {
		docOutput(rugodoc.FormatBridgePackage(pkg))
		return nil
	}

	// Mode 6: remote module (e.g. github.com/user/repo)
	if remote.IsRemoteRequire(target) {
		return docRemote(target, symbol)
	}

	return fmt.Errorf("unknown module or package: %s", target)
}

// docRemote fetches a remote module and prints its documentation.
// Recursively aggregates docs from all non-test Rugo files in the module directory.
func docRemote(target, symbol string) error {
	r := &remote.Resolver{}

	// Fetch repo and resolve entry point
	entryPath, err := r.ResolveModule(target)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", target, err)
	}

	dir := filepath.Dir(entryPath)
	fd, err := rugodoc.ExtractDirRecursive(dir, entryPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	if symbol != "" {
		doc, sig, found := rugodoc.LookupSymbol(fd, symbol)
		if !found {
			return fmt.Errorf("%s: symbol %q not found", target, symbol)
		}
		docOutput(rugodoc.FormatSymbol(doc, sig))
		return nil
	}

	docOutput(rugodoc.FormatFile(fd))
	return nil
}

// docLocalDir prints documentation for a local directory module.
// It finds the entry point Rugo file and recursively aggregates docs
// from all non-test Rugo files in the tree.
func docLocalDir(dir, symbol string) error {
	entryPath, err := compiler.FindLocalEntryPoint(dir)
	if err != nil {
		// No entry point found — still try recursive extraction without one
		entryPath = ""
	}

	fd, err := rugodoc.ExtractDirRecursive(dir, entryPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", dir, err)
	}

	if symbol != "" {
		doc, sig, found := rugodoc.LookupSymbol(fd, symbol)
		if !found {
			return fmt.Errorf("%s: symbol %q not found", dir, symbol)
		}
		docOutput(rugodoc.FormatSymbol(doc, sig))
		return nil
	}

	docOutput(rugodoc.FormatFile(fd))
	return nil
}

// docOutput prints documentation with optional ANSI color for signatures.
func docOutput(text string) {
	if os.Getenv("NO_COLOR") != "" {
		fmt.Print(text)
		return
	}
	fmt.Print(docColorize(text))
}

// ANSI color codes (Monokai-inspired)
const (
	ansiReset     = "\033[0m"
	ansiMagenta   = "\033[35m"   // keywords: def, struct, module, package
	ansiGreen     = "\033[32m"   // function/type names
	ansiYellow    = "\033[33m"   // parameters
	ansiCyan      = "\033[36m"   // return types
	ansiDim       = "\033[2m"    // doc text
	ansiBoldBlue  = "\033[1;34m" // file headers
	ansiBoldWhite = "\033[1;37m" // section titles
)

// docColorize applies syntax-aware coloring to doc output.
func docColorize(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]

		switch {
		case strings.HasPrefix(trimmed, "def "):
			lines[i] = indent + colorizeSignature(trimmed, "def")
		case strings.HasPrefix(trimmed, "struct "):
			lines[i] = indent + colorizeSignature(trimmed, "struct")
		case strings.HasPrefix(trimmed, "module "):
			lines[i] = indent + colorizeSignature(trimmed, "module")
		case strings.HasPrefix(trimmed, "package "):
			lines[i] = indent + colorizeSignature(trimmed, "package")
		case isDottedCall(trimmed):
			lines[i] = indent + colorizeDottedCall(trimmed)
		case strings.HasSuffix(trimmed, ".rugo:") || strings.HasSuffix(trimmed, ".rg:"):
			lines[i] = indent + ansiBoldBlue + trimmed + ansiReset
		case strings.HasPrefix(trimmed, "Modules (") || strings.HasPrefix(trimmed, "Bridge packages ("):
			lines[i] = indent + ansiBoldWhite + trimmed + ansiReset
		case len(indent) >= 4 && trimmed != "" && !isSignatureLine(trimmed):
			lines[i] = indent + ansiDim + trimmed + ansiReset
		}
	}
	return strings.Join(lines, "\n")
}

// colorizeSignature highlights a def/struct/module/package line with
// keyword in magenta, name in green, and params in yellow.
func colorizeSignature(line, keyword string) string {
	rest := strings.TrimPrefix(line, keyword+" ")

	// Split name from params
	if parenIdx := strings.Index(rest, "("); parenIdx >= 0 {
		name := rest[:parenIdx]
		closeIdx := strings.LastIndex(rest, ")")
		if closeIdx > parenIdx {
			params := rest[parenIdx+1 : closeIdx]
			after := rest[closeIdx+1:]
			return ansiMagenta + keyword + ansiReset + " " +
				ansiGreen + name + ansiReset +
				"(" + ansiYellow + params + ansiReset + ")" + after
		}
	}

	// No params — check for struct fields: struct Name { x, y }
	if braceIdx := strings.Index(rest, " {"); braceIdx >= 0 {
		name := rest[:braceIdx]
		fields := rest[braceIdx:]
		return ansiMagenta + keyword + ansiReset + " " +
			ansiGreen + name + ansiReset +
			ansiDim + fields + ansiReset
	}

	// Bare name
	return ansiMagenta + keyword + ansiReset + " " + ansiGreen + rest + ansiReset
}

func isSignatureLine(s string) bool {
	return strings.HasPrefix(s, "def ") ||
		strings.HasPrefix(s, "struct ") ||
		strings.HasPrefix(s, "module ") ||
		strings.HasPrefix(s, "package ")
}

// isDottedCall matches "ns.func(...)" patterns (module/bridge function signatures).
func isDottedCall(s string) bool {
	dotIdx := strings.Index(s, ".")
	if dotIdx <= 0 {
		return false
	}
	parenIdx := strings.Index(s, "(")
	return parenIdx > dotIdx
}

// colorizeDottedCall highlights "ns.func(args) -> ret" lines.
func colorizeDottedCall(s string) string {
	dotIdx := strings.Index(s, ".")
	parenIdx := strings.Index(s, "(")
	closeIdx := strings.LastIndex(s, ")")
	if dotIdx <= 0 || parenIdx <= dotIdx || closeIdx < parenIdx {
		return s
	}

	ns := s[:dotIdx]
	name := s[dotIdx+1 : parenIdx]
	params := s[parenIdx+1 : closeIdx]
	after := s[closeIdx+1:]

	result := ansiGreen + ns + ansiReset + "." +
		ansiGreen + name + ansiReset +
		"(" + ansiYellow + params + ansiReset + ")"

	// Colorize return type: " -> type"
	if strings.HasPrefix(after, " -> ") {
		ret := strings.TrimPrefix(after, " -> ")
		result += ansiDim + " -> " + ansiReset + ansiCyan + ret + ansiReset
	} else {
		result += after
	}
	return result
}

func benchAction(ctx context.Context, cmd *cli.Command) error {
	targets := cmd.Args().Slice()
	if len(targets) == 0 {
		targets = []string{"."}
	}

	if cmd.Bool("no-color") || os.Getenv("NO_COLOR") != "" {
		os.Setenv("NO_COLOR", "1")
	}

	// Collect benchmark files (_bench.rugo and _bench.rg)
	var files []string
	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", target, err)
		}
		if info.IsDir() {
			entries, err := os.ReadDir(target)
			if err != nil {
				return fmt.Errorf("reading directory %s: %w", target, err)
			}
			for _, e := range entries {
				if !e.IsDir() && compiler.IsRugoBenchFile(e.Name()) {
					files = append(files, filepath.Join(target, e.Name()))
				}
			}
		} else {
			files = append(files, target)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no benchmark files found")
	}

	for _, f := range files {
		fmt.Fprintf(os.Stderr, "=== %s ===\n", f)
		comp := &compiler.Compiler{}
		if err := comp.Run(f); err != nil {
			return err
		}
	}

	return nil
}

func testAction(ctx context.Context, cmd *cli.Command) error {
	targets := cmd.Args().Slice()
	if len(targets) == 0 {
		if info, err := os.Stat("rats"); err == nil && info.IsDir() {
			targets = []string{"rats"}
		} else {
			targets = []string{"."}
		}
	}

	// Set NO_COLOR if --no-color flag, non-interactive, or NO_COLOR already set.
	// RUGO_FORCE_COLOR is set by the parent process when it knows the terminal supports color
	// (child subprocesses have piped stderr so can't detect TTY themselves).
	if cmd.Bool("no-color") || os.Getenv("NO_COLOR") != "" {
		os.Setenv("NO_COLOR", "1")
	} else if !term.IsTerminal(int(os.Stderr.Fd())) && os.Getenv("RUGO_FORCE_COLOR") == "" {
		os.Setenv("NO_COLOR", "1")
	} else {
		os.Setenv("RUGO_FORCE_COLOR", "1")
	}

	// Per-test timeout: explicit flag > existing env var > default (30s)
	if cmd.IsSet("timeout") {
		timeout := cmd.Int("timeout")
		if timeout > 0 {
			os.Setenv("RUGO_TEST_TIMEOUT", strconv.Itoa(int(timeout)))
		} else {
			os.Unsetenv("RUGO_TEST_TIMEOUT")
		}
	} else if os.Getenv("RUGO_TEST_TIMEOUT") == "" {
		os.Setenv("RUGO_TEST_TIMEOUT", "30")
	}

	// Timing: propagate via env so subprocesses inherit it
	if cmd.Bool("timing") {
		os.Setenv("RUGO_TEST_TIMING", "1")
	}

	// Recap: propagate via env so subprocesses inherit it
	if cmd.Bool("recap") {
		os.Setenv("RUGO_TEST_RECAP", "1")
	}

	// Collect test files: _test.rugo/_test.rg files and Rugo files with inline rats blocks
	var files []string
	for _, target := range targets {
		info, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", target, err)
		}
		if info.IsDir() {
			err := filepath.WalkDir(target, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if d.IsDir() {
					if d.Name() == "fixtures" {
						return filepath.SkipDir
					}
					return nil
				}
				if compiler.IsRugoTestFile(d.Name()) {
					files = append(files, path)
				} else if compiler.IsRugoFile(d.Name()) && fileHasRatsBlocks(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return fmt.Errorf("walking directory %s: %w", target, err)
			}
		} else {
			files = append(files, target)
		}
	}

	if len(files) == 0 {
		return fmt.Errorf("no test files found")
	}

	// Single file: run directly (no subprocess overhead)
	if len(files) == 1 {
		fmt.Fprintf(os.Stderr, "=== %s ===\n", files[0])
		comp := &compiler.Compiler{TestMode: true}
		if err := comp.Run(files[0]); err != nil {
			fmt.Fprintln(os.Stderr, formatError(err.Error()))
			os.Exit(1)
		}
		return nil
	}

	// Multiple files: check concurrency setting
	jobs := cmd.Int("jobs")
	if jobs < 1 {
		jobs = 1
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find rugo binary: %w", err)
	}

	ansi := `(?:\x1b\[[0-9;]*m)*`
	summaryRe := regexp.MustCompile(ansi + `(\d+) tests, ` + ansi + `(\d+) passed` + ansi + `, ` + ansi + `(\d+) failed` + ansi + `, (\d+) skipped` + `(?: in \S+)?`)

	type fileResult struct {
		output []byte
		failed bool
	}

	results := make([]fileResult, len(files))
	showTiming := os.Getenv("RUGO_TEST_TIMING") != ""
	suiteStart := time.Now()

	if jobs == 1 {
		// Sequential: run each file with live output
		for i, f := range files {
			c := exec.Command(self, "rats", f)
			var buf bytes.Buffer
			c.Stdout = io.MultiWriter(os.Stdout, &buf)
			c.Stderr = io.MultiWriter(os.Stderr, &buf)
			if err := c.Run(); err != nil {
				results[i].failed = true
			}
			results[i].output = buf.Bytes()
		}
	} else {
		// Parallel: buffer output per file, print in order
		type asyncResult struct {
			buf  bytes.Buffer
			done chan struct{}
		}
		async := make([]asyncResult, len(files))
		for i := range async {
			async[i].done = make(chan struct{})
		}
		work := make(chan int, len(files))
		for i := range files {
			work <- i
		}
		close(work)
		var wg sync.WaitGroup
		for range jobs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range work {
					c := exec.Command(self, "rats", files[i])
					c.Stdout = &async[i].buf
					c.Stderr = &async[i].buf
					if err := c.Run(); err != nil {
						results[i].failed = true
					}
					close(async[i].done)
				}
			}()
		}
		for i := range async {
			<-async[i].done
			out := async[i].buf.Bytes()
			os.Stdout.Write(out)
			results[i].output = out
		}
	}

	// Accumulate totals and print grand summary.
	// Files that exit non-zero without producing a TAP summary (e.g. compile
	// errors) are counted as 1 failed test so the summary never lies.
	grandTests, grandPassed, grandFailed, grandSkipped := 0, 0, 0, 0
	for _, r := range results {
		if m := summaryRe.FindSubmatch(r.output); m != nil {
			t, _ := strconv.Atoi(string(m[1]))
			p, _ := strconv.Atoi(string(m[2]))
			f, _ := strconv.Atoi(string(m[3]))
			s, _ := strconv.Atoi(string(m[4]))
			grandTests += t
			grandPassed += p
			grandFailed += f
			grandSkipped += s
		} else if r.failed {
			grandTests++
			grandFailed++
		}
	}

	noColor := os.Getenv("NO_COLOR") != ""
	colorOK, colorFail, colorReset := "\033[32m", "\033[31m", "\033[0m"
	if noColor {
		colorOK, colorFail, colorReset = "", "", ""
	}
	timingTotal := ""
	if showTiming {
		timingTotal = fmt.Sprintf(" in %s", formatTestDuration(time.Since(suiteStart)))
	}
	if grandFailed > 0 {
		fmt.Fprintf(os.Stderr, "\n%d files, %d tests, %d passed, %s%d failed%s, %d skipped%s\n",
			len(files), grandTests, grandPassed, colorFail, grandFailed, colorReset, grandSkipped, timingTotal)
	} else {
		fmt.Fprintf(os.Stderr, "\n%d files, %d tests, %s%d passed%s, %d failed, %d skipped%s\n",
			len(files), grandTests, colorOK, grandPassed, colorReset, grandFailed, grandSkipped, timingTotal)
	}

	if grandFailed > 0 {
		os.Exit(1)
	}
	return nil
}

// fileHasRatsBlocks reports whether a Rugo file contains rats test blocks.
// It uses a lightweight line scan to avoid parsing the full file.
func fileHasRatsBlocks(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "rats ") || strings.HasPrefix(trimmed, "rats\t") {
			return true
		}
	}
	return false
}

// formatError colorizes an error message for terminal output.
// Respects the NO_COLOR environment variable.
func formatError(msg string) string {
	if os.Getenv("NO_COLOR") != "" || (os.Getenv("RUGO_FORCE_COLOR") == "" && !term.IsTerminal(int(os.Stderr.Fd()))) {
		return "error: " + msg
	}

	const (
		red   = "\033[31m"
		bold  = "\033[1m"
		dim   = "\033[2m"
		reset = "\033[0m"
	)

	// Colorize the "error:" prefix
	result := red + bold + "error" + reset + ": "

	// Split into main error line and optional snippet
	parts := strings.SplitN(msg, "\n", 2)
	mainLine := parts[0]

	// Bold the file:line:col prefix if present (e.g., "test.rg:2:3: ...")
	if idx := strings.Index(mainLine, ": "); idx > 0 {
		prefix := mainLine[:idx]
		// Check if it looks like a file:line reference
		if strings.Contains(prefix, ":") && !strings.Contains(prefix, " ") {
			result += bold + prefix + reset + ": " + mainLine[idx+2:]
		} else {
			result += mainLine
		}
	} else {
		result += mainLine
	}

	// Colorize source snippet if present
	if len(parts) > 1 {
		snippet := parts[1]
		var coloredLines []string
		for _, line := range strings.Split(snippet, "\n") {
			trimmed := strings.TrimSpace(line)
			if strings.HasSuffix(trimmed, "^") {
				// Caret line — show in red
				coloredLines = append(coloredLines, red+line+reset)
			} else if strings.HasPrefix(trimmed, "|") || (len(trimmed) > 0 && trimmed[len(trimmed)-1] == '|') {
				// Gutter line
				coloredLines = append(coloredLines, dim+line+reset)
			} else {
				coloredLines = append(coloredLines, line)
			}
		}
		result += "\n" + strings.Join(coloredLines, "\n")
	}

	return result
}

// formatTestDuration formats a duration in a human-friendly way for test output.
func formatTestDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
