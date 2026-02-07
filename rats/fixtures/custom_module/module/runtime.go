//go:build ignore

package hello

type Hello struct{}

// Greet returns a greeting for the given name.
func (*Hello) Greet(name string) interface{} {
	return "hello, " + name
}

// World returns the string "hello, world!".
func (*Hello) World() interface{} {
	return "hello, world!"
}
