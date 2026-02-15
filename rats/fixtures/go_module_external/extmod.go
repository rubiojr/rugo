// Package extmod demonstrates bridging functions that use types from external packages.
// bytes.Buffer is used as the external type since it's always available in the stdlib.
package extmod

import "bytes"

// NewBuffer creates a new bytes.Buffer with the given initial content.
func NewBuffer(s string) *bytes.Buffer {
	return bytes.NewBufferString(s)
}

// BufferString returns the string contents of a bytes.Buffer.
func BufferString(b *bytes.Buffer) string {
	return b.String()
}

// BufferLen returns the length of the buffer contents.
func BufferLen(b *bytes.Buffer) int {
	return b.Len()
}

// BufferWrite appends text to a buffer and returns the updated buffer.
func BufferWrite(b *bytes.Buffer, text string) *bytes.Buffer {
	b.WriteString(text)
	return b
}

// BufferApply applies a transformation function to the buffer's content
// and returns the result. Tests func+struct combo reclassification.
func BufferApply(b *bytes.Buffer, fn func(string) string) string {
	return fn(b.String())
}

// Greet is a plain function (no external types) to verify mixed packages work.
func Greet(name string) string {
	return "hi, " + name
}
