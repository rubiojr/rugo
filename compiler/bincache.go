package compiler

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

// binCacheDir returns the base directory for the binary cache.
func binCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "rugo", "bincache"), nil
}

// binCacheKey returns a hex hash key for the given Go source and go.mod content.
func binCacheKey(goSource, goMod string) string {
	h := sha256.New()
	h.Write([]byte(goSource))
	h.Write([]byte{0}) // separator
	h.Write([]byte(goMod))
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// binCacheLookup checks if a cached binary exists for the given key.
// Returns the path to the cached binary if found, empty string otherwise.
func binCacheLookup(key string) string {
	dir, err := binCacheDir()
	if err != nil {
		return ""
	}
	cached := filepath.Join(dir, key, "rugo_program")
	if _, err := os.Stat(cached); err == nil {
		return cached
	}
	return ""
}

// binCacheStore copies a compiled binary into the cache under the given key.
func binCacheStore(key, binFile string) {
	dir, err := binCacheDir()
	if err != nil {
		return
	}
	cacheEntry := filepath.Join(dir, key)
	if err := os.MkdirAll(cacheEntry, 0755); err != nil {
		return
	}
	dest := filepath.Join(cacheEntry, "rugo_program")
	data, err := os.ReadFile(binFile)
	if err != nil {
		return
	}
	os.WriteFile(dest, data, 0755)
}
