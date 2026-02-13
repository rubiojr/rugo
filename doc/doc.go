// Package doc extracts documentation from Rugo source files.
//
// It uses the compiler's ParseSource API to get AST nodes and struct metadata,
// then correlates doc comments from the raw source using line numbers. The
// attachment rule: consecutive # lines immediately before a def/struct
// declaration (no blank line gap) are attached as the doc comment.
package doc

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rubiojr/rugo/ast"
)

// FileDoc holds all extracted documentation for a single Rugo file.
type FileDoc struct {
	Path    string
	Doc     string // file-level doc (first # block before any code)
	Funcs   []FuncDoc
	Structs []StructDoc
}

// FuncDoc describes a documented function.
type FuncDoc struct {
	Name   string   // e.g. "factorial" or "Dog.bark"
	Params []string // parameter names
	Doc    string
	Line   int    // 1-based line number of the def
	Source string // relative path of the source file (set by recursive extraction)
}

// StructDoc describes a documented struct.
type StructDoc struct {
	Name   string
	Fields []string
	Doc    string
	Line   int    // 1-based line number of the struct keyword
	Source string // relative path of the source file (set by recursive extraction)
}

// ExtractFile reads a Rugo file and extracts all documentation.
func ExtractFile(path string) (*FileDoc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Extract(string(data), path), nil
}

// ExtractDir reads all Rugo files in a directory (non-recursive) and returns
// aggregated documentation. The entry file's doc becomes the top-level doc.
// Other files contribute their functions and structs.
func ExtractDir(dir, entryFile string) (*FileDoc, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	result := &FileDoc{Path: dir}

	// Extract entry file first for file-level doc
	if entryFile != "" {
		fd, err := ExtractFile(entryFile)
		if err == nil {
			result.Doc = fd.Doc
			result.Funcs = append(result.Funcs, fd.Funcs...)
			result.Structs = append(result.Structs, fd.Structs...)
		}
	}

	entryBase := ""
	if entryFile != "" {
		entryBase = filepath.Base(entryFile)
	}

	for _, e := range entries {
		if e.IsDir() || !isRugoFile(e.Name()) {
			continue
		}
		if e.Name() == entryBase {
			continue // already processed
		}
		path := filepath.Join(dir, e.Name())
		fd, err := ExtractFile(path)
		if err != nil {
			continue
		}
		result.Funcs = append(result.Funcs, fd.Funcs...)
		result.Structs = append(result.Structs, fd.Structs...)
	}

	return result, nil
}

// ExtractDirRecursive walks a directory tree and aggregates documentation
// from all non-test Rugo files. Test files (*_test.rugo) are excluded.
// The entry file's doc becomes the top-level doc.
func ExtractDirRecursive(dir, entryFile string) (*FileDoc, error) {
	result := &FileDoc{Path: dir}

	processedEntry := false
	if entryFile != "" {
		fd, err := ExtractFile(entryFile)
		if err == nil {
			result.Doc = fd.Doc
			rel, _ := filepath.Rel(dir, entryFile)
			tagSource(fd, rel)
			result.Funcs = append(result.Funcs, fd.Funcs...)
			result.Structs = append(result.Structs, fd.Structs...)
			processedEntry = true
		}
	}

	entryAbs := ""
	if entryFile != "" {
		entryAbs, _ = filepath.Abs(entryFile)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !isRugoFile(info.Name()) {
			return nil
		}
		if isTestFile(info.Name()) {
			return nil
		}
		if processedEntry {
			abs, _ := filepath.Abs(path)
			if abs == entryAbs {
				return nil
			}
		}
		fd, err := ExtractFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		tagSource(fd, rel)
		result.Funcs = append(result.Funcs, fd.Funcs...)
		result.Structs = append(result.Structs, fd.Structs...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return result, nil
}

// tagSource sets the Source field on all funcs and structs in a FileDoc.
func tagSource(fd *FileDoc, source string) {
	for i := range fd.Funcs {
		fd.Funcs[i].Source = source
	}
	for i := range fd.Structs {
		fd.Structs[i].Source = source
	}
}

// isTestFile returns true if the filename is a Rugo test file.
func isTestFile(name string) bool {
	return strings.HasSuffix(name, "_test.rugo") || strings.HasSuffix(name, "_test.rg")
}

// Extract parses raw Rugo source and returns structured documentation.
// It uses the compiler API for AST and struct metadata, and correlates
// doc comments from the raw source by line number.
func Extract(src, path string) *FileDoc {
	fd := &FileDoc{Path: path}
	lines := strings.Split(src, "\n")

	// Extract file-level doc: first comment block before any code
	fd.Doc = extractFileDoc(lines)

	// Parse with compiler to get AST + struct info
	c := &ast.Compiler{}
	prog, err := c.ParseSource(src, path)
	if err != nil {
		// Parse failed — fall back to text-based extraction for functions
		extractFuncsFromLines(fd, lines)
		return fd
	}

	// Build set of struct constructor names to exclude from func docs.
	// Struct expansion generates def Name(...) and def new(...) which
	// are not user-written functions.
	structConstructors := make(map[string]bool)
	for _, si := range prog.Structs {
		structConstructors[si.Name] = true
	}
	if len(prog.Structs) == 1 {
		structConstructors["new"] = true
	}

	// Extract function docs from AST
	for _, s := range prog.Statements {
		fn, ok := s.(*ast.FuncDef)
		if !ok {
			continue
		}
		// Skip struct constructor functions
		if structConstructors[fn.Name] {
			continue
		}
		// Check if the original source line has a method definition (Dog.bark)
		name, params, defLine := funcDocFromRawLine(lines, fn)
		// Skip private functions (underscore-prefixed)
		funcName := name
		if i := strings.LastIndex(funcName, "."); i >= 0 {
			funcName = funcName[i+1:]
		}
		if strings.HasPrefix(funcName, "_") {
			continue
		}
		fd.Funcs = append(fd.Funcs, FuncDoc{
			Name:   name,
			Params: params,
			Doc:    extractDocComment(lines, defLine),
			Line:   defLine,
		})
	}

	// Extract struct docs from preprocessor metadata
	for _, si := range prog.Structs {
		fd.Structs = append(fd.Structs, StructDoc{
			Name:   si.Name,
			Fields: si.Fields,
			Doc:    extractDocComment(lines, si.Line),
			Line:   si.Line,
		})
	}

	return fd
}

// extractFuncsFromLines extracts function documentation by scanning raw source
// lines. Used as fallback when the compiler can't parse the file (e.g. partial
// files with methods but no struct definition).
func extractFuncsFromLines(fd *FileDoc, lines []string) {
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") {
			name, params := parseDef(trimmed)
			if name == "" {
				continue
			}
			// Skip private functions (underscore-prefixed)
			funcName := name
			if j := strings.LastIndex(funcName, "."); j >= 0 {
				funcName = funcName[j+1:]
			}
			if strings.HasPrefix(funcName, "_") {
				continue
			}
			fd.Funcs = append(fd.Funcs, FuncDoc{
				Name:   name,
				Params: params,
				Doc:    extractDocComment(lines, i+1),
				Line:   i + 1,
			})
		}
		if strings.HasPrefix(trimmed, "struct ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				structName := parts[1]
				var fields []string
				for j := i + 1; j < len(lines); j++ {
					ft := strings.TrimSpace(lines[j])
					if ft == "end" {
						break
					}
					if ft != "" && !strings.HasPrefix(ft, "#") {
						fields = append(fields, ft)
					}
				}
				fd.Structs = append(fd.Structs, StructDoc{
					Name:   structName,
					Fields: fields,
					Doc:    extractDocComment(lines, i+1),
					Line:   i + 1,
				})
			}
		}
	}
}

// funcDocFromRawLine extracts the function name, params, and source line from
// the raw source, preserving method names like "Dog.bark" that the preprocessor
// rewrites. Falls back to searching by name if line mapping is off (e.g. heredocs).
func funcDocFromRawLine(lines []string, fn *ast.FuncDef) (name string, params []string, line int) {
	line = fn.StmtLine()
	if line >= 1 && line <= len(lines) {
		rawLine := strings.TrimSpace(lines[line-1])
		if strings.HasPrefix(rawLine, "def ") {
			name, params = parseDef(rawLine)
			if name != "" {
				return name, params, line
			}
		}
	}
	// Line map may be off (e.g. heredoc expansion). Search raw source for the def.
	for i, l := range lines {
		trimmed := strings.TrimSpace(l)
		if !strings.HasPrefix(trimmed, "def ") {
			continue
		}
		n, p := parseDef(trimmed)
		// Match by function name (method name may include struct prefix)
		if n == fn.Name || strings.HasSuffix(n, "."+fn.Name) {
			return n, p, i + 1
		}
	}
	return fn.Name, fn.Params, line
}

// extractFileDoc returns the first comment block before any code or blank line.
func extractFileDoc(lines []string) string {
	var block []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			block = append(block, strings.TrimPrefix(trimmed[1:], " "))
			continue
		}
		// Blank line or code line ends the file-level doc search
		break
	}
	if len(block) == 0 {
		return ""
	}
	return strings.Join(block, "\n")
}

// extractDocComment walks backwards from a declaration line to collect
// consecutive # comment lines with no blank line gap.
func extractDocComment(lines []string, declLine int) string {
	if declLine <= 1 || declLine > len(lines) {
		return ""
	}
	var block []string
	for i := declLine - 2; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "#") {
			block = append([]string{strings.TrimPrefix(trimmed[1:], " ")}, block...)
		} else {
			break
		}
	}
	return strings.Join(block, "\n")
}

// parseDef extracts the function name and parameter names from a def line.
// Handles: def foo(a, b), def Struct.method(a, b), def foo
func parseDef(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "def ") {
		return "", nil
	}
	rest := strings.TrimSpace(line[4:])

	// Split name from params
	parenIdx := strings.Index(rest, "(")
	if parenIdx < 0 {
		// No parens: def foo
		name := strings.TrimSpace(rest)
		return name, nil
	}

	name := strings.TrimSpace(rest[:parenIdx])
	paramStr := rest[parenIdx+1:]
	if closeIdx := strings.Index(paramStr, ")"); closeIdx >= 0 {
		paramStr = paramStr[:closeIdx]
	}

	var params []string
	for _, p := range strings.Split(paramStr, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			params = append(params, p)
		}
	}

	return name, params
}

// isHeredocDelim checks if a string looks like a valid heredoc delimiter.
func isHeredocDelim(s string) bool {
	for _, c := range s {
		if !((c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return len(s) > 0
}

// isRugoFile returns true if the filename has a Rugo extension (.rugo or .rg).
func isRugoFile(name string) bool {
	return strings.HasSuffix(name, ".rugo") || strings.HasSuffix(name, ".rg")
}

// LookupSymbol finds a specific function or struct by name in a FileDoc.
func LookupSymbol(fd *FileDoc, name string) (doc string, signature string, found bool) {
	for _, f := range fd.Funcs {
		if f.Name == name {
			sig := "def " + f.Name
			if len(f.Params) > 0 {
				sig += "(" + strings.Join(f.Params, ", ") + ")"
			}
			return f.Doc, sig, true
		}
	}
	for _, s := range fd.Structs {
		if s.Name == name {
			sig := "struct " + s.Name
			if len(s.Fields) > 0 {
				sig += " { " + strings.Join(s.Fields, ", ") + " }"
			}
			return s.Doc, sig, true
		}
	}
	return "", "", false
}
