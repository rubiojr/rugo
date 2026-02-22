package remote

import (
	"fmt"
	"os"
	"path/filepath"
)

// Resolver manages remote module fetching, caching, and lock file state.
type Resolver struct {
	// ModuleDir overrides the default module cache directory (~/.rugo/modules).
	ModuleDir string
	// Frozen errors if the lock file is stale or a new dependency is resolved.
	Frozen bool
	// ReadOnly prevents writing the lock file to disk.
	ReadOnly bool
	// lockFile holds parsed lock entries during compilation.
	lockFile *LockFile
	// lockFilePath is the path to the rugo.lock file.
	lockFilePath string
	// lockDirty tracks whether the lock file was modified during resolution.
	lockDirty bool
	// lockExists tracks whether the lock file existed on disk at init time.
	lockExists bool
	// hintShown tracks whether the "run rugo mod tidy" hint was printed.
	hintShown bool
	// SuppressHint prevents the "run rugo mod tidy" hint from being printed.
	SuppressHint bool
	// resolvedKeys tracks lock keys resolved during this session (for pruning).
	resolvedKeys map[string]bool
}

// InitLock sets the lock file and path for the resolver.
// Used by the update command to inject a pre-loaded lock file.
func (r *Resolver) InitLock(path string, lf *LockFile) {
	r.lockFilePath = path
	r.lockFile = lf
}

// InitLockFromDir initializes the lock file from a rugo.lock in the given directory.
func (r *Resolver) InitLockFromDir(dir string) error {
	r.lockFilePath = filepath.Join(dir, "rugo.lock")
	if _, err := os.Stat(r.lockFilePath); err == nil {
		r.lockExists = true
	}
	lf, err := ReadLockFile(r.lockFilePath)
	if err != nil {
		return err
	}
	r.lockFile = lf
	return nil
}

// WriteLockIfDirty writes the lock file to disk if it was modified.
func (r *Resolver) WriteLockIfDirty() error {
	if r.ReadOnly {
		return nil
	}
	if r.lockDirty {
		return WriteLockFile(r.lockFilePath, r.lockFile)
	}
	return nil
}

// LockDirty returns whether the lock file was modified during resolution.
func (r *Resolver) LockDirty() bool {
	return r.lockDirty
}

// trackResolved records a module key as resolved during this session.
func (r *Resolver) trackResolved(module, version string) {
	if r.resolvedKeys == nil {
		r.resolvedKeys = make(map[string]bool)
	}
	r.resolvedKeys[module+"@"+version] = true
}

// ResolvedKeys returns the set of lock keys resolved during this session.
func (r *Resolver) ResolvedKeys() map[string]bool {
	return r.resolvedKeys
}

// WriteLock writes the lock file to disk unconditionally.
func (r *Resolver) WriteLock() error {
	return WriteLockFile(r.lockFilePath, r.lockFile)
}

// LockFile returns the resolver's lock file.
func (r *Resolver) LockFile() *LockFile {
	return r.lockFile
}

// ResolveModule fetches a remote git module and returns the local
// path to its entry point Rugo file.
func (r *Resolver) ResolveModule(requirePath string) (string, error) {
	rp, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}

	cacheDir, err := r.resolveWithLock(rp)
	if err != nil {
		return "", err
	}

	return FindEntryPoint(cacheDir, rp)
}

// FetchRepo fetches a remote git repo and returns the cache directory.
// Unlike ResolveModule, it does not resolve an entry point.
func (r *Resolver) FetchRepo(requirePath string) (string, error) {
	rp, err := parseRemotePath(requirePath)
	if err != nil {
		return "", err
	}

	return r.resolveWithLock(rp)
}

// ResolveModuleOrDir fetches a remote module and tries to find a Rugo entry
// point. If no entry point is found, returns the cache directory and a nil
// error with entryPoint set to "". This allows callers to fall back to
// Go module detection without a redundant fetch.
func (r *Resolver) ResolveModuleOrDir(requirePath string) (entryPoint, cacheDir string, err error) {
	rp, err := parseRemotePath(requirePath)
	if err != nil {
		return "", "", err
	}

	cacheDir, err = r.resolveWithLock(rp)
	if err != nil {
		return "", "", err
	}

	ep, epErr := FindEntryPoint(cacheDir, rp)
	if epErr != nil {
		// Subpath support: if there's a subpath, check that directory
		if rp.Subpath != "" {
			return "", filepath.Join(cacheDir, rp.Subpath), nil
		}
		return "", cacheDir, nil
	}
	return ep, cacheDir, nil
}

// UpdateEntry re-resolves a mutable dependency, ignoring the existing
// lock entry, and updates the lock file. If module is empty, all mutable
// entries are updated.
func (r *Resolver) UpdateEntry(module string) error {
	if r.lockFile == nil {
		return fmt.Errorf("no lock file loaded")
	}

	for _, entry := range r.lockFile.Entries {
		if module != "" && entry.Module != module {
			continue
		}
		// Reconstruct the remote path. _default means "no version specified",
		// so we must not append it as a real @version suffix.
		remoteSrc := entry.Module
		if entry.Version != "_default" {
			remoteSrc += "@" + entry.Version
		}
		rp, err := parseRemotePath(remoteSrc)
		if err != nil {
			continue
		}
		// Only re-resolve mutable versions.
		if rp.isImmutable() {
			continue
		}

		// Resolve @latest to the highest semver tag before cloning.
		if isLatest(rp.Version) {
			tag, err := gitLatestTag(rp.cloneURL())
			if err != nil {
				return fmt.Errorf("updating %s@%s: %w", entry.Module, entry.Version, err)
			}
			if tag != "" {
				rp.Version = tag
			} else {
				rp.Version = ""
			}
		}

		// Clone to a unique temp dir, get SHA, atomically install.
		versionDir, err := moduleCacheDir(r.ModuleDir, rp, "")
		if err != nil {
			return err
		}
		tmpDir, err := gitCloneToTemp(rp, versionDir)
		if err != nil {
			return fmt.Errorf("updating %s@%s: %w", entry.Module, entry.Version, err)
		}
		sha, err := gitRevParseSHA(tmpDir)
		if err != nil {
			os.RemoveAll(tmpDir)
			return fmt.Errorf("updating %s@%s: %w", entry.Module, entry.Version, err)
		}

		finalDir, err := moduleCacheDir(r.ModuleDir, rp, sha)
		if err == nil {
			if installErr := atomicInstall(tmpDir, finalDir); installErr != nil {
				return installErr
			}
		} else {
			os.RemoveAll(tmpDir)
		}

		entry.SHA = sha
		r.lockDirty = true
	}

	if r.lockDirty {
		return WriteLockFile(r.lockFilePath, r.lockFile)
	}
	return nil
}

// resolveWithLock fetches a remote module, respecting the lock file.
// It returns the cache directory where the module is stored and records
// the resolved SHA in the lock file.
func (r *Resolver) resolveWithLock(rp *remotePath) (string, error) {
	moduleKey := rp.moduleKey()
	versionLabel := rp.versionLabel()

	// Ensure lock file is initialized.
	if r.lockFile == nil {
		r.lockFile = NewLockFile()
	}

	// Check the lock file for a pinned SHA.
	var lockedSHA string
	if entry := r.lockFile.Lookup(moduleKey, versionLabel); entry != nil {
		lockedSHA = entry.SHA
	}

	// Determine cache directory (uses _sha_<sha>/ for locked mutable versions).
	cacheDir, err := moduleCacheDir(r.ModuleDir, rp, lockedSHA)
	if err != nil {
		return "", err
	}

	// If locked SHA cache exists, skip fetch entirely.
	if lockedSHA != "" {
		if _, err := os.Stat(cacheDir); err == nil {
			r.trackResolved(moduleKey, versionLabel)
			return cacheDir, nil
		}
		// Locked SHA not in cache — need to fetch at that SHA.
		if r.Frozen {
			return "", fmt.Errorf("--frozen: locked module %s@%s (sha %s) not in cache", moduleKey, versionLabel, lockedSHA[:7])
		}
		if err := gitCloneAtSHA(rp, cacheDir, lockedSHA); err != nil {
			return "", err
		}
		r.trackResolved(moduleKey, versionLabel)
		return cacheDir, nil
	}

	// No lock entry — resolve normally.
	if r.Frozen {
		return "", fmt.Errorf("--frozen: no lock entry for %s@%s", moduleKey, versionLabel)
	}

	// Resolve @latest: query remote tags, pick highest semver, rewrite
	// the version so the rest of the flow clones the resolved tag.
	if isLatest(rp.Version) {
		tag, err := gitLatestTag(rp.cloneURL())
		if err != nil {
			return "", fmt.Errorf("resolving @latest for %s: %w", moduleKey, err)
		}
		if tag != "" {
			rp.Version = tag
		} else {
			// No semver tags — fall back to default branch.
			rp.Version = ""
		}
		// Recalculate cache dir with the resolved version, but keep
		// the lock entry under the "latest" label for future lookups.
		cacheDir, err = moduleCacheDir(r.ModuleDir, rp, "")
		if err != nil {
			return "", err
		}
	}

	// Hint the user to run rugo mod tidy when a lock file exists but is
	// missing an entry (stale lock). Don't hint if there's no lock file at
	// all — the user hasn't opted into dependency pinning yet.
	if !r.hintShown && !r.SuppressHint && r.lockExists {
		fmt.Fprintf(os.Stderr, "hint: run 'rugo mod tidy' to pin dependencies\n")
		r.hintShown = true
	}

	// For immutable versions, use the standard needsFetch logic.
	if rp.isImmutable() {
		if info, err := os.Stat(cacheDir); err == nil && isDirWithContent(info, cacheDir) {
			// Already cached. Record in lock file for completeness.
			sha, err := gitRevParseSHA(cacheDir)
			if err == nil {
				r.lockFile.Set(moduleKey, versionLabel, sha)
				r.lockDirty = true
			}
			r.trackResolved(moduleKey, versionLabel)
			return cacheDir, nil
		}
		if err := gitClone(rp, cacheDir); err != nil {
			return "", err
		}
		sha, err := gitRevParseSHA(cacheDir)
		if err != nil {
			return cacheDir, nil // non-fatal: lock just won't have this entry
		}
		r.lockFile.Set(moduleKey, versionLabel, sha)
		r.lockDirty = true
		r.trackResolved(moduleKey, versionLabel)
		return cacheDir, nil
	}

	// Mutable version without lock: clone to a unique temp dir, get SHA,
	// then atomically install to _sha_<sha>/.
	versionDir, err := moduleCacheDir(r.ModuleDir, rp, "")
	if err != nil {
		return "", err
	}
	tmpDir, err := gitCloneToTemp(rp, versionDir)
	if err != nil {
		return "", err
	}
	sha, err := gitRevParseSHA(tmpDir)
	if err != nil {
		// Can't get SHA — install to version-label dir as fallback.
		if installErr := atomicInstall(tmpDir, versionDir); installErr != nil {
			return "", installErr
		}
		return versionDir, nil
	}

	finalDir, err := moduleCacheDir(r.ModuleDir, rp, sha)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}

	if err := atomicInstall(tmpDir, finalDir); err != nil {
		return "", err
	}

	r.lockFile.Set(moduleKey, versionLabel, sha)
	r.lockDirty = true
	r.trackResolved(moduleKey, versionLabel)
	return finalDir, nil
}

// isDirWithContent returns true if the path is a directory with at least one entry.
// Used to detect stale empty cache directories that should be re-fetched.
func isDirWithContent(info os.FileInfo, path string) bool {
	if !info.IsDir() {
		return false
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) > 0
}
