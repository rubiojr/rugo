//go:build ignore

package helpers

import "fmt"

func rugo_go_to_string_slice(v interface{}) []string {
	arr, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("expected array, got %T", v))
	}
	r := make([]string, len(arr))
	for i, s := range arr {
		r[i] = rugo_to_string(s)
	}
	return r
}

func rugo_go_from_string_slice(v []string) interface{} {
	r := make([]interface{}, len(v))
	for i, s := range v {
		r[i] = interface{}(s)
	}
	return interface{}(r)
}
