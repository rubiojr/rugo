package compiler

import (
	"fmt"
	"strings"
)

// goWriter manages indented Go source output for the code generator.
// It encapsulates the output buffer, indentation level, and source file
// tracking for //line directives.
type goWriter struct {
	sb     strings.Builder
	indent int
	source string // current source file for //line directives
}

// Line writes an indented, formatted line (with trailing newline).
func (w *goWriter) Line(format string, args ...interface{}) {
	line := fmt.Sprintf(format, args...)
	if strings.HasSuffix(strings.TrimRight(line, "\n"), "\n") || line == "\n" {
		w.sb.WriteString(line)
		return
	}
	w.sb.WriteString(strings.Repeat("\t", w.indent) + line)
}

// Linef writes an indented, formatted line with a trailing newline appended.
func (w *goWriter) Linef(format string, args ...interface{}) {
	w.Line(format+"\n", args...)
}

// Raw writes unindented text directly to the buffer.
func (w *goWriter) Raw(s string) {
	w.sb.WriteString(s)
}

// LineDirective emits a //line directive for the current source file.
func (w *goWriter) LineDirective(line int) {
	if line > 0 && w.source != "" {
		w.sb.WriteString(fmt.Sprintf("//line %s:%d\n", w.source, line))
	}
}

// Indent increases the indentation level.
func (w *goWriter) Indent() { w.indent++ }

// Dedent decreases the indentation level.
func (w *goWriter) Dedent() { w.indent-- }

// String returns the accumulated output.
func (w *goWriter) String() string { return w.sb.String() }

// Capture runs fn while writing to a temporary buffer, then restores the
// original buffer and returns the captured output.
func (w *goWriter) Capture(fn func() error) (string, error) {
	saved := w.sb
	w.sb = strings.Builder{}
	err := fn()
	result := w.sb.String()
	w.sb = saved
	return result, err
}
