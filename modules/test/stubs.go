package testmod

import "fmt"

// Runtime helper stubs for standalone compilation and testing.

func rugo_to_string(v interface{}) string { return fmt.Sprintf("%v", v) }

func rugo_to_bool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val != ""
	case nil:
		return false
	default:
		return true
	}
}
