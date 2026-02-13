//go:build ignore

package helpers

import (
	"fmt"
	"sort"
)

func rugo_maps_keys(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		panic(fmt.Sprintf("expected hash, got %T", v))
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	sort.Strings(keys)
	r := make([]interface{}, len(keys))
	for i, k := range keys {
		r[i] = interface{}(k)
	}
	return interface{}(r)
}

func rugo_maps_values(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		panic(fmt.Sprintf("expected hash, got %T", v))
	}
	keys := make([]string, 0, len(m))
	keyMap := make(map[string]interface{})
	for k := range m {
		s := fmt.Sprintf("%v", k)
		keys = append(keys, s)
		keyMap[s] = k
	}
	sort.Strings(keys)
	r := make([]interface{}, len(keys))
	for i, k := range keys {
		r[i] = m[keyMap[k]]
	}
	return interface{}(r)
}

func rugo_maps_clone(v interface{}) interface{} {
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		panic(fmt.Sprintf("expected hash, got %T", v))
	}
	r := make(map[interface{}]interface{}, len(m))
	for k, val := range m {
		r[k] = val
	}
	return interface{}(r)
}

func rugo_maps_equal(a interface{}, b interface{}) interface{} {
	ma, ok1 := a.(map[interface{}]interface{})
	mb, ok2 := b.(map[interface{}]interface{})
	if !ok1 || !ok2 {
		return interface{}(false)
	}
	if len(ma) != len(mb) {
		return interface{}(false)
	}
	for k, va := range ma {
		vb, ok := mb[k]
		if !ok {
			return interface{}(false)
		}
		if fmt.Sprintf("%v", va) != fmt.Sprintf("%v", vb) {
			return interface{}(false)
		}
	}
	return interface{}(true)
}
