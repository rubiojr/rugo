package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/compiler"
	"github.com/rubiojr/rugo/remote"
	"github.com/urfave/cli/v3"
)

const (
	toolPrefix  = "rugo-"
	coreRemote  = "github.com/rubiojr/rugo"
	coreSubpath = "tools"
)

// toolsDir returns the path to the tools directory, creating it if needed.
// Checks RUGO_TOOLS_DIR env var first, falls back to ~/.rugo/tools.
func toolsDir() (string, error) {
	dir := os.Getenv("RUGO_TOOLS_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		dir = filepath.Join(home, ".rugo", "tools")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating tools directory: %w", err)
	}
	return dir, nil
}

// toolBinPath returns the full path for an installed tool binary.
func toolBinPath(name string) (string, error) {
	dir, err := toolsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, toolPrefix+name), nil
}

// installedToolCommands scans ~/.rugo/tools/ and returns cli.Command entries
// for each installed tool so they appear in the top-level help output.
func installedToolCommands() []*cli.Command {
	dir, err := toolsDir()
	if err != nil {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var cmds []*cli.Command
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".desc") {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, toolPrefix) {
			continue
		}
		toolName := strings.TrimPrefix(name, toolPrefix)
		usage := readToolDesc(filepath.Join(dir, name+".desc"))
		cmds = append(cmds, &cli.Command{
			Name:            toolName,
			Usage:           usage,
			SkipFlagParsing: true,
			Action: func(ctx context.Context, cmd *cli.Command) error {
				return execTool(cmd.Name, cmd.Args().Slice())
			},
		})
	}
	return cmds
}

// readToolDesc reads a .desc sidecar file. Returns a fallback if missing.
func readToolDesc(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "Installed tool extension"
	}
	desc := strings.TrimSpace(string(data))
	if desc == "" {
		return "Installed tool extension"
	}
	return desc
}

// extractToolDesc parses a "# tool: <description>" header from a Rugo source file.
func extractToolDesc(entryPoint string) string {
	data, err := os.ReadFile(entryPoint)
	if err != nil {
		return ""
	}
	for _, line := range strings.SplitN(string(data), "\n", 20) {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# tool:") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# tool:"))
		}
		// Stop at first non-comment, non-blank line
		if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			break
		}
	}
	return ""
}

// writeToolDesc writes a .desc sidecar file next to the binary.
func writeToolDesc(binPath, desc string) {
	if desc == "" {
		return
	}
	os.WriteFile(binPath+".desc", []byte(desc), 0644)
}

// execTool looks for an installed tool and runs it.
// Returns an error if the tool is not found.
func execTool(name string, args []string) error {
	binPath, err := toolBinPath(name)
	if err != nil {
		return err
	}
	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("tool not found: %s", name)
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		return err
	}
	os.Exit(0)
	return nil
}

// installLocalTool builds a local tool directory and installs it.
func installLocalTool(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("path not found: %s", path)
	}
	if !info.IsDir() {
		return fmt.Errorf("expected a directory: %s", path)
	}

	name := filepath.Base(absPath)
	entryPoint := findEntryPoint(absPath, name)
	if entryPoint == "" {
		return fmt.Errorf("no main.rugo or %s.rugo found in %s", name, path)
	}

	binPath, err := toolBinPath(name)
	if err != nil {
		return err
	}

	comp := &compiler.Compiler{}
	if err := comp.Build(entryPoint, binPath); err != nil {
		return fmt.Errorf("building tool: %w", err)
	}

	writeToolDesc(binPath, extractToolDesc(entryPoint))
	fmt.Printf("installed %s → %s\n", name, binPath)
	return nil
}

// installRemoteTool fetches a remote module and installs it as a tool.
func installRemoteTool(remotePath string) error {
	resolver := &remote.Resolver{SuppressHint: true}
	if err := resolver.InitLockFromDir("."); err != nil {
		// No lock file — that's fine for tool installs
		resolver.InitLock("", nil)
	}

	cacheDir, err := resolver.FetchRepo(remotePath)
	if err != nil {
		return fmt.Errorf("fetching %s: %w", remotePath, err)
	}

	// Determine tool name from the last path segment (before @version)
	clean := remotePath
	if i := strings.LastIndex(clean, "@"); i > 0 {
		clean = clean[:i]
	}
	parts := strings.Split(clean, "/")
	name := parts[len(parts)-1]

	// Find entry point in the fetched directory
	// For subpath modules (e.g. github.com/user/repo/tools/linter), check the subpath
	entryDir := cacheDir
	rp := parseRemoteForTool(remotePath)
	if rp.subpath != "" {
		entryDir = filepath.Join(cacheDir, rp.subpath)
	}

	entryPoint := findEntryPoint(entryDir, name)
	if entryPoint == "" {
		return fmt.Errorf("no main.rugo or %s.rugo found in fetched module", name)
	}

	binPath, err := toolBinPath(name)
	if err != nil {
		return err
	}

	comp := &compiler.Compiler{}
	if err := comp.Build(entryPoint, binPath); err != nil {
		return fmt.Errorf("building tool: %w", err)
	}

	writeToolDesc(binPath, extractToolDesc(entryPoint))
	fmt.Printf("installed %s → %s\n", name, binPath)
	return nil
}

// installCore fetches the official rugo repo and installs all tools from tools/.
func installCore(version string) error {
	remotePath := coreRemote
	if version != "" {
		remotePath += "@" + version
	}

	resolver := &remote.Resolver{SuppressHint: true}
	resolver.InitLock("", nil)

	cacheDir, err := resolver.FetchRepo(remotePath)
	if err != nil {
		return fmt.Errorf("fetching core tools: %w", err)
	}

	toolsPath := filepath.Join(cacheDir, coreSubpath)
	entries, err := os.ReadDir(toolsPath)
	if err != nil {
		return fmt.Errorf("reading tools directory: %w", err)
	}

	installed := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(toolsPath, name)
		entryPoint := findEntryPoint(dir, name)
		if entryPoint == "" {
			continue
		}

		binPath, err := toolBinPath(name)
		if err != nil {
			return err
		}

		comp := &compiler.Compiler{}
		if err := comp.Build(entryPoint, binPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to build %s: %v\n", name, err)
			continue
		}

		writeToolDesc(binPath, extractToolDesc(entryPoint))
		fmt.Printf("installed %s → %s\n", name, binPath)
		installed++
	}

	if installed == 0 {
		return fmt.Errorf("no tools found in core")
	}
	fmt.Printf("%d core tool(s) installed.\n", installed)
	return nil
}

// findEntryPoint finds the entry Rugo file in a directory.
// Checks: <name>.rugo, main.rugo
func findEntryPoint(dir, name string) string {
	// 1. <name>.rugo
	if p := filepath.Join(dir, name+".rugo"); fileExists(p) {
		return p
	}
	// 2. main.rugo
	if p := filepath.Join(dir, "main.rugo"); fileExists(p) {
		return p
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

type toolRemotePath struct {
	subpath string
}

// parseRemoteForTool extracts subpath info from a remote path.
func parseRemoteForTool(path string) toolRemotePath {
	clean := path
	if i := strings.LastIndex(clean, "@"); i > 0 {
		clean = clean[:i]
	}
	parts := strings.Split(clean, "/")
	if len(parts) > 3 {
		return toolRemotePath{subpath: strings.Join(parts[3:], "/")}
	}
	return toolRemotePath{}
}

func toolInstallAction(ctx context.Context, cmd *cli.Command) error {
	args := cmd.Args().Slice()
	if len(args) == 0 {
		return fmt.Errorf("usage: rugo tool install <path | remote-module | core>")
	}

	target := args[0]

	// Magic: "core" or "core@version"
	if target == "core" || strings.HasPrefix(target, "core@") {
		version := ""
		if strings.HasPrefix(target, "core@") {
			version = target[5:]
		}
		return installCore(version)
	}

	// Remote module (contains a dot in the first segment, like github.com/...)
	if remote.IsRemoteRequire(target) {
		return installRemoteTool(target)
	}

	// Local path
	return installLocalTool(target)
}

func toolListAction(ctx context.Context, cmd *cli.Command) error {
	dir, err := toolsDir()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading tools directory: %w", err)
	}

	found := 0
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".desc") {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, toolPrefix) {
			continue
		}
		toolName := strings.TrimPrefix(name, toolPrefix)
		desc := readToolDesc(filepath.Join(dir, name+".desc"))
		fmt.Printf("%-20s %s\n", toolName, desc)
		found++
	}

	if found == 0 {
		fmt.Println("No tools installed. Run 'rugo tool install core' to get started.")
	}
	return nil
}

func toolRemoveAction(ctx context.Context, cmd *cli.Command) error {
	if cmd.NArg() < 1 {
		return fmt.Errorf("usage: rugo tool remove <name>")
	}

	name := cmd.Args().First()
	binPath, err := toolBinPath(name)
	if err != nil {
		return err
	}

	if _, err := os.Stat(binPath); err != nil {
		return fmt.Errorf("tool not installed: %s", name)
	}

	if err := os.Remove(binPath); err != nil {
		return fmt.Errorf("removing tool: %w", err)
	}
	os.Remove(binPath + ".desc") // clean up sidecar

	fmt.Printf("removed %s\n", name)
	return nil
}
