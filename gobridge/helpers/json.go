//go:build ignore

package helpers

import "fmt"

func rugo_json_prepare(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[fmt.Sprintf("%v", k)] = rugo_json_prepare(v)
		}
		return m
	case []interface{}:
		r := make([]interface{}, len(val))
		for i, e := range val {
			r[i] = rugo_json_prepare(e)
		}
		return r
	default:
		return v
	}
}

func rugo_json_to_rugo(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[interface{}]interface{}, len(val))
		for k, v := range val {
			m[interface{}(k)] = rugo_json_to_rugo(v)
		}
		return m
	case []interface{}:
		r := make([]interface{}, len(val))
		for i, e := range val {
			r[i] = rugo_json_to_rugo(e)
		}
		return r
	default:
		return v
	}
}
