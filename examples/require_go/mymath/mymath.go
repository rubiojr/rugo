package mymath

import "math"

func Add(a int, b int) int {
	return a + b
}

func Multiply(a int, b int) int {
	return a * b
}

func Sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func IsPositive(n int) bool {
	return n > 0
}
