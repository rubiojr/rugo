//go:build ignore

package fmtmod

import "fmt"

// --- fmt module ---

type Fmt struct{}

// Sprintf formats a string using Go's fmt.Sprintf.
func (*Fmt) Sprintf(format string, args ...interface{}) interface{} {
	return fmt.Sprintf(format, args...)
}

// Printf prints a formatted string to stdout using Go's fmt.Printf.
func (*Fmt) Printf(format string, args ...interface{}) interface{} {
	fmt.Printf(format, args...)
	return nil
}
