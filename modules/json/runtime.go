package jsonmod

import (
	"encoding/json"
	"fmt"
	"math"
)

// --- json module ---

type JSON struct{}

func (*JSON) Parse(s string) interface{} {
	var raw interface{}
	if err := json.Unmarshal([]byte(s), &raw); err != nil {
		if se, ok := err.(*json.SyntaxError); ok {
			panic(fmt.Sprintf("json.parse: invalid JSON at position %d", se.Offset))
		}
		panic(fmt.Sprintf("json.parse: invalid JSON: %v", err))
	}
	return convertJSON(raw)
}

func (*JSON) Encode(val interface{}) interface{} {
	b, err := json.Marshal(prepareJSON(val))
	if err != nil {
		panic(fmt.Sprintf("json.encode: %v", err))
	}
	return string(b)
}

func (*JSON) Pretty(val interface{}) interface{} {
	b, err := json.MarshalIndent(prepareJSON(val), "", "  ")
	if err != nil {
		panic(fmt.Sprintf("json.pretty: %v", err))
	}
	return string(b)
}

// prepareJSON converts Rugo types back to standard Go types for json.Marshal.
func prepareJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, child := range val {
			m[fmt.Sprintf("%v", k)] = prepareJSON(child)
		}
		return m
	case []interface{}:
		for i, child := range val {
			val[i] = prepareJSON(child)
		}
		return val
	default:
		return v
	}
}

// convertJSON recursively converts Go json.Unmarshal types to Rugo-friendly types.
// Whole-number float64 values become int so they work naturally with conv.to_s etc.
func convertJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[interface{}]interface{}, len(val))
		for k, child := range val {
			m[k] = convertJSON(child)
		}
		return m
	case []interface{}:
		for i, child := range val {
			val[i] = convertJSON(child)
		}
		return val
	case float64:
		if val == math.Trunc(val) && !math.IsInf(val, 0) && !math.IsNaN(val) {
			return int(val)
		}
		return val
	default:
		return v
	}
}
