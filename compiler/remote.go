package compiler

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// isRemoteRequire returns true if the require path looks like a git URL
// (e.g. "github.com/user/repo"). Local paths like "helpers" or "lib/math"
// never have a dot in the first segment. Relative paths starting with
// "." or ".." are always local.
func isRemoteRequire(path string) bool {
	// Relative paths are always local
	if strings.HasPrefix(path, ".") {
		return false
	}
	// Strip version suffix for detection
	clean := path
	if i := strings.Index(clean, "@"); i > 0 {
		clean = clean[:i]
	}
	parts := strings.SplitN(clean, "/", 2)
	if len(parts) < 2 {
		return false
	}
	host := parts[0]
	// localhost (with optional port) is remote (for testing with git daemon)
	if host == "localhost" || strings.HasPrefix(host, "localhost:") {
		return true
	}
	return strings.Contains(host, ".")
}

// remotePath holds the parsed components of a remote require path.
type remotePath struct {
	// Host is the git host (e.g. "github.com").
	Host string
	// Owner is the repository owner (e.g. "rubiojr").
	Owner string
	// Repo is the repository name (e.g. "rugo-utils").
	Repo string
	// Version is the git ref: tag, branch, or commit SHA. Empty means default branch.
	Version string
	// Subpath is the optional path within the repo (e.g. "str_utils" for monorepos).
	Subpath string
}

// cloneURL returns the clone URL for the repository.
// Uses http:// for localhost (testing with local servers), https:// for everything else.
func (r *remotePath) cloneURL() string {
	if r.Host == "localhost" || strings.HasPrefix(r.Host, "localhost:") {
		return fmt.Sprintf("http://%s/%s/%s.git", r.Host, r.Owner, r.Repo)
	}
	return fmt.Sprintf("https://%s/%s/%s.git", r.Host, r.Owner, r.Repo)
}

// versionLabel returns the version string for cache paths.
// Empty version is stored as "_default" to avoid empty directory names.
func (r *remotePath) versionLabel() string {
	if r.Version == "" {
		return "_default"
	}
	return r.Version
}

// isImmutable returns true if the version is a tag (v-prefixed) or a commit SHA.
// Immutable versions are cached forever; mutable ones (branches) are re-fetched.
func (r *remotePath) isImmutable() bool {
	if r.Version == "" {
		return false
	}
	if strings.HasPrefix(r.Version, "v") {
		return true
	}
	return isSHA(r.Version)
}

// defaultNamespace returns the namespace derived from the repo name,
// sanitized for use as a Rugo identifier (hyphens become underscores).
func (r *remotePath) defaultNamespace() string {
	name := r.Repo
	if r.Subpath != "" {
		name = filepath.Base(r.Subpath)
	}
	return sanitizeNamespace(name)
}

// isSHA returns true if s looks like a git commit SHA (7-40 hex chars).
var shaPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

func isSHA(s string) bool {
	return shaPattern.MatchString(s)
}

// sanitizeNamespace converts a string into a valid Rugo identifier
// by replacing hyphens with underscores.
func sanitizeNamespace(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// parseRemotePath parses a remote require path into its components.
//
// Examples:
//
//	"github.com/user/repo"            → host, owner, repo, "", ""
//	"github.com/user/repo@v1.0.0"    → host, owner, repo, "v1.0.0", ""
//	"github.com/user/repo/sub@main"  → host, owner, repo, "main", "sub"
func parseRemotePath(path string) (*remotePath, error) {
	// Split off @version suffix
	version := ""
	if i := strings.LastIndex(path, "@"); i > 0 {
		version = path[i+1:]
		path = path[:i]
	}

	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("remote require path must be host/owner/repo, got %q", path)
	}

	r := &remotePath{
		Host:    parts[0],
		Owner:   parts[1],
		Repo:    parts[2],
		Version: version,
	}

	if len(parts) > 3 {
		r.Subpath = strings.Join(parts[3:], "/")
	}

	return r, nil
}

// moduleCacheDir returns the local cache directory for a remote module.
// Uses ModuleDir from the compiler if set, then RUGO_MODULE_DIR env var,
// then falls back to ~/.rugo/modules/.
//
// For locked mutable versions, uses _sha_<sha>/ to store each resolved
// commit independently. Immutable versions use their version label directly.
func (c *Compiler) moduleCacheDir(r *remotePath, lockedSHA string) (string, error) {
	base := c.ModuleDir
	if base == "" {
		base = os.Getenv("RUGO_MODULE_DIR")
	}
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		base = filepath.Join(home, ".rugo", "modules")
	}

	versionDir := r.versionLabel()
	if lockedSHA != "" && !r.isImmutable() {
		versionDir = "_sha_" + lockedSHA
	}

	return filepath.Join(base, r.Host, r.Owner, r.Repo, versionDir), nil
}

// moduleCacheBase returns the base cache directory for module lookups
// (without the version subdirectory).
func (c *Compiler) moduleCacheBase() (string, error) {
	base := c.ModuleDir
	if base == "" {
		base = os.Getenv("RUGO_MODULE_DIR")
	}
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot determine home directory: %w", err)
		}
		base = filepath.Join(home, ".rugo", "modules")
	}
	return base, nil
}

// needsFetch returns true if the module needs to be (re-)fetched.
// Immutable versions (tags, SHAs) are only fetched once.
// Mutable versions (branches, default) are always re-fetched.
func needsFetch(cacheDir string, r *remotePath) bool {
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		return true
	}
	return !r.isImmutable()
}

// gitClone clones the repository into dest.
// Tries a shallow clone first; falls back to a full clone if the server
// doesn't support shallow capabilities (e.g. dumb HTTP).
func gitClone(r *remotePath, dest string) error {
	// Remove existing directory for re-fetch (mutable versions)
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("cleaning cache directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	args := []string{"clone", "--depth", "1"}
	if r.Version != "" {
		args = append(args, "--branch", r.Version)
	}
	args = append(args, r.cloneURL(), dest)

	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Shallow clone may fail with dumb HTTP servers; retry without --depth
		os.RemoveAll(dest)
		args = []string{"clone"}
		if r.Version != "" {
			args = append(args, "--branch", r.Version)
		}
		args = append(args, r.cloneURL(), dest)
		cmd = exec.Command("git", args...)
		cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone %s: %s", r.cloneURL(), strings.TrimSpace(string(output)))
		}
	}
	return nil
}

// gitRevParseSHA returns the full commit SHA for HEAD in the given git repo directory.
func gitRevParseSHA(repoDir string) (string, error) {
	cmd := exec.Command("git", "-C", repoDir, "rev-parse", "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse HEAD in %s: %w", repoDir, err)
	}
	return strings.TrimSpace(string(output)), nil
}

// gitCloneAtSHA clones a repo and checks out a specific commit SHA.
// Used when the lock file pins a mutable version to a known SHA.
func gitCloneAtSHA(r *remotePath, dest, sha string) error {
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("cleaning cache directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}

	// Clone without branch, then checkout the specific SHA.
	args := []string{"clone", r.cloneURL(), dest}
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %s", r.cloneURL(), strings.TrimSpace(string(output)))
	}

	// Checkout the locked SHA.
	cmd = exec.Command("git", "-C", dest, "checkout", sha)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	output, err = cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(dest)
		return fmt.Errorf("git checkout %s: %s", sha, strings.TrimSpace(string(output)))
	}
	return nil
}

// findEntryPoint locates the main .rg file in a cloned module directory.
//
// Resolution order (at root or subpath):
//  1. <subpath>.rg at the repo root (flat-file subpath, e.g. client.rg)
//  2. <name>.rg in the subpath directory (repo name or subpath basename)
//  3. main.rg
//  4. Exactly one .rg file
func findEntryPoint(cacheDir string, r *remotePath) (string, error) {
	dir := cacheDir
	name := r.Repo
	if r.Subpath != "" {
		// Flat-file subpath: check for <root>/<subpath>.rg first
		flatCandidate := filepath.Join(cacheDir, r.Subpath+".rg")
		if fileExists(flatCandidate) {
			return flatCandidate, nil
		}
		dir = filepath.Join(cacheDir, r.Subpath)
		name = filepath.Base(r.Subpath)
	}

	// 1. <name>.rg
	candidate := filepath.Join(dir, name+".rg")
	if fileExists(candidate) {
		return candidate, nil
	}

	// 2. main.rg
	candidate = filepath.Join(dir, "main.rg")
	if fileExists(candidate) {
		return candidate, nil
	}

	// 3. Exactly one .rg file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading module directory %s: %w", dir, err)
	}
	var rgFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".rg") {
			rgFiles = append(rgFiles, filepath.Join(dir, e.Name()))
		}
	}
	if len(rgFiles) == 1 {
		return rgFiles[0], nil
	}

	if len(rgFiles) == 0 {
		return "", fmt.Errorf("no .rg files found in module %s/%s/%s", r.Host, r.Owner, r.Repo)
	}
	hint := name + ".rg or main.rg, or use 'with' to select specific modules"
	return "", fmt.Errorf("cannot determine entry point for module %s/%s/%s: found %d .rg files (add a %s)", r.Host, r.Owner, r.Repo, len(rgFiles), hint)
}

// resolveRemoteWithLock fetches a remote module, respecting the lock file.
// It returns the cache directory where the module is stored and records
// the resolved SHA in the lock file.
func (c *Compiler) resolveRemoteWithLock(r *remotePath) (string, error) {
	moduleKey := r.moduleKey()
	versionLabel := r.versionLabel()

	// Ensure lock file is initialized (may be called outside Compile).
	if c.lockFile == nil {
		c.lockFile = NewLockFile()
	}

	// Check the lock file for a pinned SHA.
	var lockedSHA string
	if entry := c.lockFile.Lookup(moduleKey, versionLabel); entry != nil {
		lockedSHA = entry.SHA
	}

	// Determine cache directory (uses _sha_<sha>/ for locked mutable versions).
	cacheDir, err := c.moduleCacheDir(r, lockedSHA)
	if err != nil {
		return "", err
	}

	// If locked SHA cache exists, skip fetch entirely.
	if lockedSHA != "" {
		if _, err := os.Stat(cacheDir); err == nil {
			return cacheDir, nil
		}
		// Locked SHA not in cache — need to fetch at that SHA.
		if c.Frozen {
			return "", fmt.Errorf("--frozen: locked module %s@%s (sha %s) not in cache", moduleKey, versionLabel, lockedSHA[:7])
		}
		if err := gitCloneAtSHA(r, cacheDir, lockedSHA); err != nil {
			return "", err
		}
		return cacheDir, nil
	}

	// No lock entry — resolve normally.
	if c.Frozen {
		return "", fmt.Errorf("--frozen: no lock entry for %s@%s", moduleKey, versionLabel)
	}

	// For immutable versions, use the standard needsFetch logic.
	if r.isImmutable() {
		if _, err := os.Stat(cacheDir); err == nil {
			// Already cached. Record in lock file for completeness.
			sha, err := gitRevParseSHA(cacheDir)
			if err == nil {
				c.lockFile.Set(moduleKey, versionLabel, sha)
				c.lockDirty = true
			}
			return cacheDir, nil
		}
		if err := gitClone(r, cacheDir); err != nil {
			return "", err
		}
		sha, err := gitRevParseSHA(cacheDir)
		if err != nil {
			return cacheDir, nil // non-fatal: lock just won't have this entry
		}
		c.lockFile.Set(moduleKey, versionLabel, sha)
		c.lockDirty = true
		return cacheDir, nil
	}

	// Mutable version without lock: clone to a temp dir, get SHA, then move to _sha_<sha>/.
	tmpDir, err := c.moduleCacheDir(r, "")
	if err != nil {
		return "", err
	}
	if err := gitClone(r, tmpDir); err != nil {
		return "", err
	}
	sha, err := gitRevParseSHA(tmpDir)
	if err != nil {
		// Can't get SHA — fall back to the temp dir without lock.
		return tmpDir, nil
	}

	// Move to the SHA-keyed directory.
	finalDir, err := c.moduleCacheDir(r, sha)
	if err != nil {
		return tmpDir, nil
	}

	if finalDir != tmpDir {
		// If SHA dir already exists, remove the temp clone.
		if _, err := os.Stat(finalDir); err == nil {
			os.RemoveAll(tmpDir)
		} else {
			os.MkdirAll(filepath.Dir(finalDir), 0755)
			if err := os.Rename(tmpDir, finalDir); err != nil {
				// Rename failed (cross-device?), keep tmpDir.
				return tmpDir, nil
			}
		}
	}

	c.lockFile.Set(moduleKey, versionLabel, sha)
	c.lockDirty = true
	return finalDir, nil
}

// ResolveRemoteModule fetches a remote git module and returns the local
// path to its entry point .rg file.
func (c *Compiler) ResolveRemoteModule(requirePath string) (string, error) {
	r, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}

	cacheDir, err := c.resolveRemoteWithLock(r)
	if err != nil {
		return "", err
	}

	return findEntryPoint(cacheDir, r)
}

// FetchRemoteRepo fetches a remote git repo and returns the cache directory.
// Unlike ResolveRemoteModule, it does not resolve an entry point.
func (c *Compiler) FetchRemoteRepo(requirePath string) (string, error) {
	r, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}

	return c.resolveRemoteWithLock(r)
}

// UpdateLockEntry re-resolves a mutable dependency, ignoring the existing
// lock entry, and updates the lock file. If module is empty, all mutable
// entries are updated.
func (c *Compiler) UpdateLockEntry(module string) error {
	if c.lockFile == nil {
		return fmt.Errorf("no lock file loaded")
	}

	for _, entry := range c.lockFile.Entries {
		if module != "" && entry.Module != module {
			continue
		}
		// Reconstruct the remote path. _default means "no version specified",
		// so we must not append it as a real @version suffix.
		remoteSrc := entry.Module
		if entry.Version != "_default" {
			remoteSrc += "@" + entry.Version
		}
		r, err := parseRemotePath(remoteSrc)
		if err != nil {
			continue
		}
		// Only re-resolve mutable versions.
		if r.isImmutable() {
			continue
		}

		// Clone fresh to the version-label dir (not SHA dir).
		tmpDir, err := c.moduleCacheDir(r, "")
		if err != nil {
			return err
		}
		if err := gitClone(r, tmpDir); err != nil {
			return fmt.Errorf("updating %s@%s: %w", entry.Module, entry.Version, err)
		}
		sha, err := gitRevParseSHA(tmpDir)
		if err != nil {
			return fmt.Errorf("updating %s@%s: %w", entry.Module, entry.Version, err)
		}

		// Move to SHA-keyed directory.
		finalDir, err := c.moduleCacheDir(r, sha)
		if err == nil && finalDir != tmpDir {
			if _, err := os.Stat(finalDir); err == nil {
				os.RemoveAll(tmpDir)
			} else {
				os.MkdirAll(filepath.Dir(finalDir), 0755)
				os.Rename(tmpDir, finalDir)
			}
		}

		entry.SHA = sha
		c.lockDirty = true
	}

	if c.lockDirty {
		return WriteLockFile(c.lockFilePath, c.lockFile)
	}
	return nil
}

// remoteDefaultNamespace returns the default namespace for a remote require path.
func remoteDefaultNamespace(requirePath string) (string, error) {
	r, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}
	return r.defaultNamespace(), nil
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
