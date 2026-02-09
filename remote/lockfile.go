package remote

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LockEntry records the resolved commit SHA for a single remote module.
type LockEntry struct {
	// Module is the repo path without version (e.g. "github.com/user/repo").
	Module string
	// Version is the requested version label (e.g. "v1.0.0", "main", "_default").
	Version string
	// SHA is the resolved full commit SHA.
	SHA string
}

// lockKey returns the deduplication key for a lock entry.
// A module can appear multiple times with different versions.
func (e *LockEntry) lockKey() string {
	return e.Module + "@" + e.Version
}

// LockFile holds all lock entries for a project.
type LockFile struct {
	Entries []*LockEntry
	index   map[string]*LockEntry // lockKey → entry
}

// NewLockFile creates an empty lock file.
func NewLockFile() *LockFile {
	return &LockFile{index: make(map[string]*LockEntry)}
}

// Lookup finds the lock entry for a given module and version label.
// Returns nil if not found.
func (lf *LockFile) Lookup(module, version string) *LockEntry {
	if lf.index == nil {
		return nil
	}
	return lf.index[module+"@"+version]
}

// Set adds or updates a lock entry.
func (lf *LockFile) Set(module, version, sha string) {
	if lf.index == nil {
		lf.index = make(map[string]*LockEntry)
	}
	entry := &LockEntry{Module: module, Version: version, SHA: sha}
	key := entry.lockKey()
	if existing, ok := lf.index[key]; ok {
		existing.SHA = sha
		return
	}
	lf.Entries = append(lf.Entries, entry)
	lf.index[key] = entry
}

// ReadLockFile reads a rugo.lock file. Returns an empty LockFile if the
// file does not exist.
func ReadLockFile(path string) (*LockFile, error) {
	lf := NewLockFile()

	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return lf, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading lock file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("rugo.lock:%d: expected 3 fields (module version sha), got %d", lineNum, len(fields))
		}

		lf.Set(fields[0], fields[1], fields[2])
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading lock file: %w", err)
	}

	return lf, nil
}

// WriteLockFile writes the lock file to disk.
func WriteLockFile(path string, lf *LockFile) error {
	if len(lf.Entries) == 0 {
		// No entries — remove the lock file if it exists.
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing empty lock file: %w", err)
		}
		return nil
	}

	var sb strings.Builder
	sb.WriteString("# rugo.lock — auto-generated, do not edit\n")
	for _, e := range lf.Entries {
		fmt.Fprintf(&sb, "%s %s %s\n", e.Module, e.Version, e.SHA)
	}

	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("writing lock file: %w", err)
	}
	return nil
}
