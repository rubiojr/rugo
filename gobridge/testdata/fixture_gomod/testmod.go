package testmod

func Greet(name string) string {
	return "hello, " + name
}

func Add(a int, b int) int {
	return a + b
}

func IsEven(n int) bool {
	return n%2 == 0
}

// Blocked: pointer parameter
func WithPointer(p *string) string {
	return *p
}

// Blocked: returns a channel
func MakeChan() chan int {
	return make(chan int)
}

// unexported: should not appear in bridge
func helper() string {
	return "hidden"
}
