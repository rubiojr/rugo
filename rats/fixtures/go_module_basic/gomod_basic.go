package gomod_basic

import "strings"

// Foo
// Bar
func Greet(name string) string {
	return "hello, " + name
}

func Add(a int, b int) int {
	return a + b
}

func Upper(s string) string {
	return strings.ToUpper(s)
}
