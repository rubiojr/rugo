package compiler

import (
	"crypto/sha256"
	"fmt"
	"github.com/rubiojr/rugo/ast"
	"os"
	"path/filepath"
	"strings"
)

// processEmbeds validates embed paths and emits //go:embed directives with
// package-level string variables. Each embedded file is staged as
// embeds/<hash>_<basename> to avoid name collisions.
func (g *codeGen) processEmbeds(embeds []*ast.EmbedStmt, file *GoFile) error {
	if g.disableEmbed {
		e := embeds[0]
		return &ast.UserError{Msg: fmt.Sprintf("%s:%d: embed is not supported in eval.run() (no files to embed); use eval.file() instead", g.sourceFile, e.SourceLine)}
	}

	// Pre-pass: check for duplicate aliases before touching the filesystem.
	seenAliases := make(map[string]int) // alias → source line
	for _, e := range embeds {
		if prev, exists := seenAliases[e.Alias]; exists {
			return &ast.UserError{Msg: fmt.Sprintf("%s:%d: embed alias %q already used at line %d", g.sourceFile, e.SourceLine, e.Alias, prev)}
		}
		seenAliases[e.Alias] = e.SourceLine
	}

	for _, e := range embeds {
		absPath, err := resolveEmbedPath(e.Path, g.sourceFile, e.SourceFile)
		if err != nil {
			return &ast.UserError{Msg: fmt.Sprintf("%s:%d: %s", g.sourceFile, e.SourceLine, err)}
		}

		// Verify the file exists and is a regular file.
		info, err := os.Stat(absPath)
		if err != nil {
			return &ast.UserError{Msg: fmt.Sprintf("%s:%d: embed %q: %s", g.sourceFile, e.SourceLine, e.Path, err)}
		}
		if info.IsDir() {
			return &ast.UserError{Msg: fmt.Sprintf("%s:%d: embed %q: is a directory, not a file", g.sourceFile, e.SourceLine, e.Path)}
		}

		// Generate a unique staged filename: <short_hash>_<basename>
		stagedName := stagedEmbedName(absPath, e.Alias)
		g.embedFiles[stagedName] = absPath

		// Emit: //go:embed embeds/<stagedName>
		//        var _embed_<alias> string
		goVarName := "_embed_" + e.Alias
		file.Decls = append(file.Decls, GoRawDecl{
			Code: fmt.Sprintf("//go:embed embeds/%s\nvar %s string\n", stagedName, goVarName),
		})
	}
	if len(embeds) > 0 {
		file.Decls = append(file.Decls, GoBlankLine{})
	}
	return nil
}

// embedInitStmts returns Go statements that assign embed vars to Rugo-visible
// package-level variables in main(). e.g.: config = interface{}(_embed_config)
func embedInitStmts(embeds []*ast.EmbedStmt) []GoStmt {
	var stmts []GoStmt
	for _, e := range embeds {
		goVarName := "_embed_" + e.Alias
		stmts = append(stmts, GoAssignStmt{
			Target: e.Alias,
			Op:     "=",
			Value:  GoCastExpr{Type: "interface{}", Value: GoIdentExpr{Name: goVarName}},
		})
	}
	return stmts
}

// resolveEmbedPath resolves an embed path relative to the source file that
// declares it. Returns the absolute path and validates that it does not escape
// the source file's directory tree (Go-style restriction).
func resolveEmbedPath(embedPath, mainSourceFile, stmtSourceFile string) (string, error) {
	if embedPath == "" {
		return "", fmt.Errorf("embed path cannot be empty")
	}
	if filepath.IsAbs(embedPath) {
		return "", fmt.Errorf("embed path %q must be relative (absolute paths are not allowed)", embedPath)
	}

	// Determine the base directory: use the statement's own source file if it
	// came from a require'd file, otherwise use the main source file.
	baseFile := mainSourceFile
	if stmtSourceFile != "" {
		baseFile = stmtSourceFile
	}

	// Resolve baseFile to absolute. This handles both relative display paths
	// (e.g., "test.rugo") and absolute paths correctly.
	absBase, err := filepath.Abs(baseFile)
	if err != nil {
		return "", fmt.Errorf("resolving source path: %w", err)
	}
	baseDir := filepath.Dir(absBase)

	// Resolve the embed path relative to the base directory.
	resolved := filepath.Join(baseDir, embedPath)
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolving embed path %q: %w", embedPath, err)
	}

	// Security: ensure the resolved path is within baseDir or a subdirectory.
	if !isSubPath(baseDir, absResolved) {
		return "", fmt.Errorf("embed path %q escapes source directory (must be in same directory or subdirectory)", embedPath)
	}

	return absResolved, nil
}

// isSubPath checks if target is equal to or under the base directory.
func isSubPath(base, target string) bool {
	// Ensure base ends with separator for proper prefix matching.
	basePrefix := base + string(filepath.Separator)
	return target == base || strings.HasPrefix(target, basePrefix)
}

// stagedEmbedName generates a unique filename for staging: <hash8>_<basename>
func stagedEmbedName(absPath, alias string) string {
	h := sha256.Sum256([]byte(absPath))
	prefix := fmt.Sprintf("%x", h[:4])
	return prefix + "_" + filepath.Base(absPath)
}

// stageEmbedFiles copies embedded files into tmpDir/embeds/ for go build.
func stageEmbedFiles(tmpDir string, embedFiles map[string]string) error {
	if len(embedFiles) == 0 {
		return nil
	}
	embedDir := filepath.Join(tmpDir, "embeds")
	if err := os.MkdirAll(embedDir, 0755); err != nil {
		return fmt.Errorf("creating embeds dir: %w", err)
	}
	for stagedName, srcPath := range embedFiles {
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return fmt.Errorf("reading embed file %s: %w", srcPath, err)
		}
		dst := filepath.Join(embedDir, stagedName)
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("staging embed file %s: %w", stagedName, err)
		}
	}
	return nil
}

// embedCacheHash returns a hash incorporating all embedded file contents,
// suitable for extending the binary cache key.
func embedCacheHash(embedFiles map[string]string) string {
	if len(embedFiles) == 0 {
		return ""
	}
	h := sha256.New()
	// Sort keys for deterministic ordering.
	keys := make([]string, 0, len(embedFiles))
	for k := range embedFiles {
		keys = append(keys, k)
	}
	sortStrings(keys)
	for _, k := range keys {
		data, err := os.ReadFile(embedFiles[k])
		if err != nil {
			// If we can't read, include the path to invalidate cache.
			h.Write([]byte(embedFiles[k]))
			continue
		}
		h.Write([]byte(k))
		h.Write([]byte{0})
		h.Write(data)
		h.Write([]byte{0})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
