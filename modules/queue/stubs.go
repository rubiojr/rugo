package queuemod

// Runtime helper stubs for standalone compilation and testing.

func rugo_to_int(v interface{}) int {
	switch n := v.(type) {
	case int:
		return n
	case float64:
		return int(n)
	default:
		panic("expected an integer")
	}
}
