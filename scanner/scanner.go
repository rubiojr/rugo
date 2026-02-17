// Package scanner provides string-boundary-aware scanning for the Rugo
// preprocessor. It encapsulates the tracking of double-quoted, single-quoted,
// and backtick string literals plus escape sequences, eliminating the need
// for every preprocessor function to re-implement this logic.
package scanner

import "strings"

// closingKind tracks which type of string delimiter was just closed.
type closingKind byte

const (
	noClosing       closingKind = iota
	closingDouble               // just closed a "..." string
	closingSingle               // just closed a '...' string
	closingBacktick             // just closed a `...` expression
)

// CodeScanner iterates byte-by-byte over source text, tracking string
// literal boundaries (double-quoted, single-quoted, backtick) and escape
// sequences. Callers check InString() instead of maintaining their own
// inDouble/inSingle/escaped flags.
//
// InString() returns true for the entire string span including both
// opening and closing delimiters, matching the preprocessor's convention
// of skipping all bytes that are part of string literals.
type CodeScanner struct {
	src     string
	pos     int
	line    int
	inDbl   bool
	inSgl   bool
	inBt    bool
	escaped bool
	closing closingKind // set when a closing delimiter is processed
}

// New creates a CodeScanner for the given source text.
// Call Next() to advance to the first byte.
func New(src string) *CodeScanner {
	return &CodeScanner{src: src, pos: -1, line: 1}
}

// Next advances to the next byte, updating string/escape state.
// Returns the byte and true, or (0, false) at end of input.
func (s *CodeScanner) Next() (byte, bool) {
	s.closing = noClosing
	s.pos++
	if s.pos >= len(s.src) {
		return 0, false
	}
	ch := s.src[s.pos]
	if ch == '\n' {
		s.line++
	}

	if s.escaped {
		s.escaped = false
		return ch, true
	}
	if ch == '\\' && (s.inDbl || s.inSgl) {
		s.escaped = true
		return ch, true
	}
	if ch == '"' && !s.inSgl && !s.inBt {
		if s.inDbl {
			s.closing = closingDouble
		}
		s.inDbl = !s.inDbl
	} else if ch == '\'' && !s.inDbl && !s.inBt {
		if s.inSgl {
			s.closing = closingSingle
		}
		s.inSgl = !s.inSgl
	} else if ch == '`' && !s.inDbl && !s.inSgl {
		if s.inBt {
			s.closing = closingBacktick
		}
		s.inBt = !s.inBt
	}

	return ch, true
}

// InString reports whether the current position is inside a string literal
// (double-quoted, single-quoted, or backtick), including both opening and
// closing delimiters.
func (s *CodeScanner) InString() bool {
	return s.inDbl || s.inSgl || s.inBt || s.closing != noClosing
}

// InDoubleString reports whether the current position is inside a
// double-quoted string literal.
func (s *CodeScanner) InDoubleString() bool { return s.inDbl || s.closing == closingDouble }

// InSingleString reports whether the current position is inside a
// single-quoted string literal.
func (s *CodeScanner) InSingleString() bool { return s.inSgl || s.closing == closingSingle }

// InBacktick reports whether the current position is inside a backtick
// expression.
func (s *CodeScanner) InBacktick() bool { return s.inBt || s.closing == closingBacktick }

// InCode reports whether the current position is outside all string literals.
func (s *CodeScanner) InCode() bool { return !s.InString() }

// Pos returns the current byte offset (the position of the last byte
// returned by Next). Returns -1 before the first call to Next.
func (s *CodeScanner) Pos() int { return s.pos }

// Line returns the current 1-based line number.
func (s *CodeScanner) Line() int { return s.line }

// Src returns the full source text being scanned.
func (s *CodeScanner) Src() string { return s.src }

// Peek returns the next byte without advancing, or (0, false) at end.
func (s *CodeScanner) Peek() (byte, bool) {
	if s.pos+1 >= len(s.src) {
		return 0, false
	}
	return s.src[s.pos+1], true
}

// LookingAt checks if src[pos:] starts with the given prefix.
// Useful for multi-character token detection (e.g., "||", "or", "fn(").
func (s *CodeScanner) LookingAt(prefix string) bool {
	return strings.HasPrefix(s.src[s.pos:], prefix)
}

// Skip advances past n bytes without returning them. String/escape state
// is updated for each skipped byte. Returns the number of bytes actually
// skipped (may be less than n at end of input).
func (s *CodeScanner) Skip(n int) int {
	skipped := 0
	for i := 0; i < n; i++ {
		if _, ok := s.Next(); !ok {
			break
		}
		skipped++
	}
	return skipped
}

// IsOpenBracket reports whether ch is an opening bracket/paren/brace.
func IsOpenBracket(ch byte) bool {
	return ch == '(' || ch == '[' || ch == '{'
}

// IsCloseBracket reports whether ch is a closing bracket/paren/brace.
func IsCloseBracket(ch byte) bool {
	return ch == ')' || ch == ']' || ch == '}'
}

// FindTopLevel scans s for a byte matching pred at bracket depth 0,
// outside all string literals. Returns the byte offset or -1.
//
// This covers the common pattern used by findCompoundOp, findDestructAssign,
// findDoAssignment, findTopLevelOr, findTopLevelPipes, etc.
func FindTopLevel(s string, pred func(ch byte, pos int, src string) bool) int {
	depth := 0
	sc := New(s)
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			continue
		}
		if IsOpenBracket(ch) {
			depth++
		} else if IsCloseBracket(ch) {
			depth--
		}
		if depth == 0 && pred(ch, sc.Pos(), s) {
			return sc.Pos()
		}
	}
	return -1
}

// FindAllTopLevel is like FindTopLevel but returns all matching positions.
func FindAllTopLevel(s string, pred func(ch byte, pos int, src string) bool) []int {
	var positions []int
	depth := 0
	sc := New(s)
	for ch, ok := sc.Next(); ok; ch, ok = sc.Next() {
		if sc.InString() {
			continue
		}
		if IsOpenBracket(ch) {
			depth++
		} else if IsCloseBracket(ch) {
			depth--
		}
		if depth == 0 && pred(ch, sc.Pos(), s) {
			positions = append(positions, sc.Pos())
		}
	}
	return positions
}

// IsInsideString reports whether byte offset pos in s falls inside a
// string literal. This matches the original preprocessor behavior: it
// checks the string state just before pos (scanning bytes 0..pos-1),
// so opening delimiters return false and closing delimiters return true.
func IsInsideString(s string, pos int) bool {
	sc := New(s)
	for i := 0; i < pos; i++ {
		if _, ok := sc.Next(); !ok {
			return false
		}
	}
	// Use raw state (inDbl/inSgl/inBt) which reflects whether a string
	// has been opened but not yet closed, without the closing-delimiter
	// correction that InString() applies.
	return sc.inDbl || sc.inSgl || sc.inBt
}
