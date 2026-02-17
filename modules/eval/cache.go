package evalmod

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rubiojr/rugo/ast"
	"github.com/rubiojr/rugo/compiler"
	"github.com/rubiojr/rugo/gobridge"
	"github.com/rubiojr/rugo/modules"
	"github.com/rubiojr/rugo/parser"
	"github.com/rubiojr/rugo/preprocess"
	"github.com/rubiojr/rugo/remote"
	"github.com/rubiojr/rugo/util"
)

// goModTemplate declares the module path and all dependencies needed to build
// programs that import the compiler package.
const goModTemplate = `module github.com/rubiojr/rugo

go 1.22

require modernc.org/scanner v1.3.0

require modernc.org/token v1.1.0 // indirect
`

// EnsureCompilerCache writes the full compiler source tree to
// ~/.cache/rugo/rugoeval/<hash>/ if it doesn't already exist.
// Returns the absolute path to the cache dir.
func EnsureCompilerCache() (string, error) {
	hash := compilerCacheHash()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "rugo", "rugoeval", hash)
	sentinel := filepath.Join(cacheDir, ".complete")
	if _, err := os.Stat(sentinel); err == nil {
		return cacheDir, nil
	}

	// Write ast/ sources (reuse ast package's embedded files).
	if err := writeEmbedFS(ast.Sources, filepath.Join(cacheDir, "ast")); err != nil {
		return "", fmt.Errorf("writing ast sources: %w", err)
	}

	// Write parser/ source.
	parserDir := filepath.Join(cacheDir, "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		return "", fmt.Errorf("creating parser cache dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(parserDir, "parser.go"), parser.Source, 0644); err != nil {
		return "", fmt.Errorf("writing parser.go: %w", err)
	}

	// Write compiler/ sources (includes templates/ subdirectory).
	if err := writeEmbedFS(compiler.Sources, filepath.Join(cacheDir, "compiler")); err != nil {
		return "", fmt.Errorf("writing compiler sources: %w", err)
	}

	// Write gobridge/ sources.
	if err := writeEmbedFS(gobridge.Sources, filepath.Join(cacheDir, "gobridge")); err != nil {
		return "", fmt.Errorf("writing gobridge sources: %w", err)
	}

	// Write modules/ sources (module.go + all subdirectories).
	if err := writeEmbedFS(modules.Sources, filepath.Join(cacheDir, "modules")); err != nil {
		return "", fmt.Errorf("writing modules sources: %w", err)
	}

	// Write remote/ sources.
	if err := writeEmbedFS(remote.Sources, filepath.Join(cacheDir, "remote")); err != nil {
		return "", fmt.Errorf("writing remote sources: %w", err)
	}

	// Write preprocess/ sources.
	if err := writeEmbedFS(preprocess.Sources, filepath.Join(cacheDir, "preprocess")); err != nil {
		return "", fmt.Errorf("writing preprocess sources: %w", err)
	}

	// Write util/ sources.
	if err := writeEmbedFS(util.Sources, filepath.Join(cacheDir, "util")); err != nil {
		return "", fmt.Errorf("writing util sources: %w", err)
	}

	// Write go.mod.
	if err := os.WriteFile(filepath.Join(cacheDir, "go.mod"), []byte(goModTemplate), 0644); err != nil {
		return "", fmt.Errorf("writing go.mod: %w", err)
	}

	// Mark complete.
	if err := os.WriteFile(sentinel, []byte("ok"), 0644); err != nil {
		return "", fmt.Errorf("writing sentinel: %w", err)
	}

	return cacheDir, nil
}

// writeEmbedFS writes all files from an embed.FS to the given directory,
// preserving subdirectory structure.
func writeEmbedFS(fsys embed.FS, destDir string) error {
	return fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, path)
		if d.IsDir() {
			return os.MkdirAll(dest, 0755)
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", path, err)
		}
		return os.WriteFile(dest, data, 0644)
	})
}

// compilerCacheHash returns a short hash of all embedded sources for cache keying.
func compilerCacheHash() string {
	h := sha256.New()
	hashFS(h, ast.Sources)
	h.Write(parser.Source)
	hashFS(h, compiler.Sources)
	hashFS(h, gobridge.Sources)
	hashFS(h, modules.Sources)
	hashFS(h, remote.Sources)
	hashFS(h, preprocess.Sources)
	hashFS(h, util.Sources)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// hashFS walks an embed.FS and writes all file contents to the hash.
func hashFS(h interface{ Write([]byte) (int, error) }, fsys embed.FS) {
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		h.Write(data)
		return nil
	})
}
