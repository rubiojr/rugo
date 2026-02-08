package webmod

import "fmt"

// Type conversion stubs for standalone compilation.
// These mirror the runtime helpers provided by the generated program.

func rugo_to_string(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func rugo_to_int(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		n := 0
		fmt.Sscanf(val, "%d", &n)
		return n
	default:
		return 0
	}
}

func rugo_to_bool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case nil:
		return false
	case int:
		return val != 0
	case string:
		return val != ""
	default:
		return true
	}
}

// Dispatch map stub for standalone compilation.
var rugo_web_dispatch = map[string]func(interface{}) interface{}{}
