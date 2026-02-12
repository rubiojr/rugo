package ast

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/rubiojr/rugo/parser"
)

//go:embed nodes.go walker.go preprocess.go parse.go
var Sources embed.FS

// goModTemplate is the go.mod for the cached AST module.
// It declares the same module path as rugo so the replace directive works,
// and only depends on modernc.org/scanner (the parser's sole external dep).
const goModTemplate = `module github.com/rubiojr/rugo

go 1.22

require modernc.org/scanner v1.3.0

require modernc.org/token v1.1.0 // indirect
`

// EnsureCache writes the AST parser sources to ~/.cache/rugo/rugoast/<hash>/
// if they don't already exist. Returns the absolute path to the cache dir.
// The cache provides a minimal Go module that satisfies:
//
//	import "github.com/rubiojr/rugo/ast"
//	import "github.com/rubiojr/rugo/parser"
func EnsureCache() (string, error) {
	hash := cacheHash()
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "rugo", "rugoast", hash)
	sentinel := filepath.Join(cacheDir, ".complete")
	if _, err := os.Stat(sentinel); err == nil {
		return cacheDir, nil
	}

	// Write ast/ source files.
	astDir := filepath.Join(cacheDir, "ast")
	if err := os.MkdirAll(astDir, 0755); err != nil {
		return "", fmt.Errorf("creating ast cache dir: %w", err)
	}
	entries, err := fs.ReadDir(Sources, ".")
	if err != nil {
		return "", fmt.Errorf("reading embedded ast sources: %w", err)
	}
	for _, e := range entries {
		data, err := fs.ReadFile(Sources, e.Name())
		if err != nil {
			return "", fmt.Errorf("reading embedded %s: %w", e.Name(), err)
		}
		if err := os.WriteFile(filepath.Join(astDir, e.Name()), data, 0644); err != nil {
			return "", fmt.Errorf("writing %s: %w", e.Name(), err)
		}
	}

	// Write parser/ source.
	parserDir := filepath.Join(cacheDir, "parser")
	if err := os.MkdirAll(parserDir, 0755); err != nil {
		return "", fmt.Errorf("creating parser cache dir: %w", err)
	}
	if err := os.WriteFile(filepath.Join(parserDir, "parser.go"), parser.Source, 0644); err != nil {
		return "", fmt.Errorf("writing parser.go: %w", err)
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

// cacheHash returns a short hash of the embedded sources for cache keying.
func cacheHash() string {
	h := sha256.New()
	entries, _ := fs.ReadDir(Sources, ".")
	for _, e := range entries {
		data, _ := fs.ReadFile(Sources, e.Name())
		h.Write(data)
	}
	h.Write(parser.Source)
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}
