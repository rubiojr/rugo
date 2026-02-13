//go:build ignore

package helpers

import "fmt"

func rugo_slices_contains(v interface{}, target interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("expected array, got %T", v))
	}
	ts := fmt.Sprintf("%v", target)
	for _, e := range arr {
		if fmt.Sprintf("%v", e) == ts {
			return interface{}(true)
		}
	}
	return interface{}(false)
}

func rugo_slices_index(v interface{}, target interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("expected array, got %T", v))
	}
	ts := fmt.Sprintf("%v", target)
	for i, e := range arr {
		if fmt.Sprintf("%v", e) == ts {
			return interface{}(i)
		}
	}
	return interface{}(-1)
}

func rugo_slices_reverse(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("expected array, got %T", v))
	}
	r := make([]interface{}, len(arr))
	for i, e := range arr {
		r[len(arr)-1-i] = e
	}
	return interface{}(r)
}

func rugo_slices_compact(v interface{}) interface{} {
	arr, ok := v.([]interface{})
	if !ok {
		panic(fmt.Sprintf("expected array, got %T", v))
	}
	if len(arr) == 0 {
		return interface{}([]interface{}{})
	}
	r := []interface{}{arr[0]}
	for i := 1; i < len(arr); i++ {
		if fmt.Sprintf("%v", arr[i]) != fmt.Sprintf("%v", arr[i-1]) {
			r = append(r, arr[i])
		}
	}
	return interface{}(r)
}
