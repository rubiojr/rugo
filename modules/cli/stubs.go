package climod

import "fmt"

// Runtime helper stubs — these functions are provided by the Rugo runtime in
// generated programs. They are defined here so runtime.go compiles as a normal
// Go package and can be tested directly.

func rugo_to_string(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func rugo_to_int(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		panic(fmt.Sprintf("cannot convert %T to int", v))
	}
}

func rugo_to_float(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	default:
		panic(fmt.Sprintf("cannot convert %T to float", v))
	}
}

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

// rugo_cli_dispatch is the command→handler dispatch map. In generated programs,
// the compiler populates this with user-defined handler functions. Here it
// starts empty so the module compiles standalone.
var rugo_cli_dispatch = map[string]func(interface{}) interface{}{}
