// Package doc extracts documentation from Rugo source files.
//
// It works on raw .rg source before preprocessing, since the compiler's
// stripComments phase destroys comments. The extraction rule is simple:
// consecutive # lines immediately before a def/struct declaration (no blank
// line gap) are attached as the doc comment for that declaration.
package doc

import (
	"os"
	"path/filepath"
	"strings"
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
	Line   int // 1-based line number of the def
}

// StructDoc describes a documented struct.
type StructDoc struct {
	Name   string
	Fields []string
	Doc    string
	Line   int // 1-based line number of the struct keyword
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

// Extract parses raw Rugo source and returns structured documentation.
func Extract(src, path string) *FileDoc {
	fd := &FileDoc{Path: path}
	lines := strings.Split(src, "\n")

	// State tracking
	var commentBlock []string
	commentStart := 0
	seenCode := false  // true once we've seen any non-comment, non-blank line
	inHeredoc := false // skip # lines inside heredocs
	heredocEnd := ""   // closing delimiter for current heredoc
	inStruct := false  // inside a struct block
	structName := ""   // current struct name
	var structFields []string
	structDoc := ""
	structLine := 0

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)

		// Track heredoc state to avoid treating # in heredocs as comments
		if inHeredoc {
			if trimmed == heredocEnd {
				inHeredoc = false
				heredocEnd = ""
			}
			continue
		}

		// Detect heredoc start: something = <<DELIM or <<-DELIM
		if idx := strings.Index(line, "<<"); idx >= 0 {
			rest := strings.TrimPrefix(line[idx+2:], "-")
			delim := strings.TrimSpace(rest)
			if len(delim) > 0 && isHeredocDelim(delim) {
				inHeredoc = true
				heredocEnd = delim
				seenCode = true
				commentBlock = nil
				continue
			}
		}

		// Inside struct block: collect fields
		if inStruct {
			if trimmed == "end" {
				fd.Structs = append(fd.Structs, StructDoc{
					Name:   structName,
					Fields: structFields,
					Doc:    structDoc,
					Line:   structLine,
				})
				inStruct = false
				structFields = nil
				commentBlock = nil
				continue
			}
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				structFields = append(structFields, trimmed)
			}
			continue
		}

		// Comment line
		if strings.HasPrefix(trimmed, "#") {
			commentText := strings.TrimPrefix(trimmed[1:], " ")
			if len(commentBlock) == 0 {
				commentStart = lineNum
			}
			commentBlock = append(commentBlock, commentText)
			continue
		}

		// Blank line: potential file-level doc boundary
		if trimmed == "" {
			if len(commentBlock) > 0 && !seenCode {
				// First comment block before any code = file-level doc
				fd.Doc = strings.Join(commentBlock, "\n")
				seenCode = true // prevent re-assignment
			}
			// Blank line breaks attachment
			commentBlock = nil
			continue
		}

		// Non-comment, non-blank line: this is code
		if !seenCode && len(commentBlock) > 0 {
			// First comment block before any code = file-level doc
			fd.Doc = strings.Join(commentBlock, "\n")
		}
		seenCode = true

		// Check for struct declaration
		if strings.HasPrefix(trimmed, "struct ") {
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				inStruct = true
				structName = parts[1]
				structDoc = strings.Join(commentBlock, "\n")
				structLine = lineNum
				_ = commentStart
				commentBlock = nil
				continue
			}
		}

		// Check for function declaration
		if strings.HasPrefix(trimmed, "def ") {
			name, params := parseDef(trimmed)
			if name != "" {
				fd.Funcs = append(fd.Funcs, FuncDoc{
					Name:   name,
					Params: params,
					Doc:    strings.Join(commentBlock, "\n"),
					Line:   lineNum,
				})
			}
			commentBlock = nil
			continue
		}

		// Any other code line: discard pending comment block
		commentBlock = nil
	}

	return fd
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
