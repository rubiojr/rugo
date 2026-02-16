//go:build ignore

package helpers

import "fmt"

// rugo_to_byte_slice converts a Rugo value to []byte.
// Strings are converted directly. Arrays of ints are converted element-wise.
func rugo_to_byte_slice(v interface{}) []byte {
	switch val := v.(type) {
	case string:
		return []byte(val)
	case []byte:
		return val
	case []interface{}:
		buf := make([]byte, len(val))
		for i, elem := range val {
			buf[i] = byte(rugo_to_int(elem))
		}
		return buf
	case nil:
		return nil
	default:
		panic(fmt.Sprintf("cannot convert %T to []byte", v))
	}
}
