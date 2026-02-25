package compiler

import (
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const binCacheMaxBytes = 10 * 1024 * 1024 * 1024 // 10 GB

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

// binCacheLookup checks if a compressed cached binary exists for the given key.
// Returns the path to the cached .gz file if found, empty string otherwise.
// Touches the file to update its LRU timestamp.
func binCacheLookup(key string) string {
	dir, err := binCacheDir()
	if err != nil {
		return ""
	}
	cached := filepath.Join(dir, key+".gz")
	if _, err := os.Stat(cached); err == nil {
		now := time.Now()
		os.Chtimes(cached, now, now)
		return cached
	}
	return ""
}

// binCacheDecompress reads a gzip-compressed binary from gzPath and writes it
// to destPath with executable permissions.
func binCacheDecompress(gzPath, destPath string) error {
	f, err := os.Open(gzPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	out, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, gr)
	return err
}

// binCacheStore compresses and stores a compiled binary in the cache, then
// runs LRU eviction if the cache exceeds the size cap.
func binCacheStore(key, binFile string) {
	dir, err := binCacheDir()
	if err != nil {
		return
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	dest := filepath.Join(dir, key+".gz")
	data, err := os.ReadFile(binFile)
	if err != nil {
		return
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	gw, err := gzip.NewWriterLevel(f, gzip.BestSpeed)
	if err != nil {
		f.Close()
		return
	}
	if _, err := gw.Write(data); err != nil {
		gw.Close()
		f.Close()
		return
	}
	gw.Close()
	f.Close()

	binCacheEvict(dir)
}

// binCacheEvict removes the oldest entries until the cache is under the size cap.
func binCacheEvict(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	type entry struct {
		path    string
		size    int64
		modTime time.Time
	}

	var files []entry
	var totalSize int64
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		files = append(files, entry{path: path, size: info.Size(), modTime: info.ModTime()})
		totalSize += info.Size()
	}

	if totalSize <= binCacheMaxBytes {
		return
	}

	// Sort oldest first.
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	for _, f := range files {
		if totalSize <= binCacheMaxBytes {
			break
		}
		os.Remove(f.path)
		totalSize -= f.size
	}
}
