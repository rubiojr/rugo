package remote

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// IsRemoteRequire returns true if the require path looks like a git URL
// (e.g. "github.com/user/repo"). Local paths like "helpers" or "lib/math"
// never have a dot in the first segment. Relative paths starting with
// "." or ".." are always local.
func IsRemoteRequire(path string) bool {
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

// moduleKey returns the module path (host/owner/repo) for a remotePath,
// used as the lock file module identifier.
func (r *remotePath) moduleKey() string {
	return r.Host + "/" + r.Owner + "/" + r.Repo
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

// DefaultNamespace returns the default namespace for a remote require path.
func DefaultNamespace(requirePath string) (string, error) {
	r, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}
	return r.defaultNamespace(), nil
}

// moduleCacheDir returns the local cache directory for a remote module.
func moduleCacheDir(moduleDir string, r *remotePath, lockedSHA string) (string, error) {
	base, err := moduleCacheBase(moduleDir)
	if err != nil {
		return "", err
	}

	versionDir := r.versionLabel()
	if lockedSHA != "" && !r.isImmutable() {
		versionDir = "_sha_" + lockedSHA
	}

	return filepath.Join(base, r.Host, r.Owner, r.Repo, versionDir), nil
}

// moduleCacheBase returns the base cache directory for module lookups
// (without the version subdirectory).
func moduleCacheBase(moduleDir string) (string, error) {
	if moduleDir != "" {
		return moduleDir, nil
	}
	if envDir := os.Getenv("RUGO_MODULE_DIR"); envDir != "" {
		return envDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".rugo", "modules"), nil
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

// FindEntryPoint locates the main Rugo file in a cloned module directory.
//
// Resolution order (at root or subpath):
//  1. <subpath>.rugo or <subpath>.rg at the repo root (flat-file subpath)
//  2. <name>.rugo or <name>.rg in the subpath directory
//  3. main.rugo or main.rg
//  4. Exactly one Rugo source file
func FindEntryPoint(cacheDir string, r *remotePath) (string, error) {
	dir := cacheDir
	name := r.Repo
	if r.Subpath != "" {
		// Flat-file subpath: check for <root>/<subpath>.rugo or .rg first
		if found := findRugoFile(filepath.Join(cacheDir, r.Subpath)); found != "" {
			return found, nil
		}
		dir = filepath.Join(cacheDir, r.Subpath)
		name = filepath.Base(r.Subpath)
	}

	// 1. <name>.rugo or <name>.rg
	if found := findRugoFile(filepath.Join(dir, name)); found != "" {
		return found, nil
	}

	// 2. main.rugo or main.rg
	if found := findRugoFile(filepath.Join(dir, "main")); found != "" {
		return found, nil
	}

	// 3. Exactly one Rugo source file
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("reading module directory %s: %w", dir, err)
	}
	var rgFiles []string
	for _, e := range entries {
		if !e.IsDir() && isRugoFile(e.Name()) {
			rgFiles = append(rgFiles, filepath.Join(dir, e.Name()))
		}
	}
	if len(rgFiles) == 1 {
		return rgFiles[0], nil
	}

	if len(rgFiles) == 0 {
		return "", fmt.Errorf("no Rugo source files found in module %s/%s/%s", r.Host, r.Owner, r.Repo)
	}
	hint := name + ".rugo or main.rugo, or use 'with' to select specific modules"
	return "", fmt.Errorf("cannot determine entry point for module %s/%s/%s: found %d Rugo files (add a %s)", r.Host, r.Owner, r.Repo, len(rgFiles), hint)
}

// isRugoFile returns true if the filename has a Rugo extension (.rugo or .rg).
func isRugoFile(name string) bool {
	return strings.HasSuffix(name, ".rugo") || strings.HasSuffix(name, ".rg")
}

// findRugoFile looks for a Rugo source file, preferring .rugo over .rg.
func findRugoFile(basePath string) string {
	if fileExists(basePath + ".rugo") {
		return basePath + ".rugo"
	}
	if fileExists(basePath + ".rg") {
		return basePath + ".rg"
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
