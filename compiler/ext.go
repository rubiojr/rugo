package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// RugoExt is the preferred file extension for Rugo source files.
	RugoExt = ".rugo"
	// DeprecatedExt is the old file extension, still supported but deprecated.
	DeprecatedExt = ".rg"
)

// IsRugoFile returns true if the filename has a Rugo extension (.rugo or .rg).
func IsRugoFile(name string) bool {
	return strings.HasSuffix(name, RugoExt) || strings.HasSuffix(name, DeprecatedExt)
}

// IsRugoTestFile returns true if the filename is a Rugo test file (_test.rugo or _test.rg).
func IsRugoTestFile(name string) bool {
	return strings.HasSuffix(name, "_test"+RugoExt) || strings.HasSuffix(name, "_test"+DeprecatedExt)
}

// IsRugoBenchFile returns true if the filename is a Rugo benchmark file (_bench.rugo or _bench.rg).
func IsRugoBenchFile(name string) bool {
	return strings.HasSuffix(name, "_bench"+RugoExt) || strings.HasSuffix(name, "_bench"+DeprecatedExt)
}

// TrimRugoExt removes the Rugo file extension (.rugo or .rg) from a filename.
func TrimRugoExt(name string) string {
	if strings.HasSuffix(name, RugoExt) {
		return strings.TrimSuffix(name, RugoExt)
	}
	return strings.TrimSuffix(name, DeprecatedExt)
}

// WarnDeprecatedExt prints a deprecation warning to stderr if the file uses
// the old .rg extension.
func WarnDeprecatedExt(filename string) {
	if !strings.HasSuffix(filename, DeprecatedExt) {
		return
	}
	base := filepath.Base(filename)
	newBase := strings.TrimSuffix(base, DeprecatedExt) + RugoExt
	fmt.Fprintf(os.Stderr, "warning: .rg extension is deprecated, rename %s to %s\n", base, newBase)
}

// FindRugoFile looks for a Rugo source file, preferring .rugo over .rg.
// Given a base path without extension, returns the path if found, empty string otherwise.
func FindRugoFile(basePath string) string {
	if fileExists(basePath + RugoExt) {
		return basePath + RugoExt
	}
	if fileExists(basePath + DeprecatedExt) {
		return basePath + DeprecatedExt
	}
	return ""
}
