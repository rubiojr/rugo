package convmod

import (
	"fmt"
	"strconv"
)

// --- conv module ---

type Conv struct{}

// rugoTypeName returns a human-friendly Rugo type name for a value.
func rugoTypeName(val interface{}) string {
	switch val.(type) {
	case int:
		return "integer"
	case float64:
		return "float"
	case string:
		return "string"
	case bool:
		return "boolean"
	case nil:
		return "nil"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "hash"
	default:
		return fmt.Sprintf("%T", val)
	}
}

func (*Conv) ToI(val interface{}) interface{} {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("conv.to_i: cannot convert %q to integer", v))
		}
		return n
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		panic(fmt.Sprintf("conv.to_i: cannot convert %s to integer", rugoTypeName(val)))
	}
}

func (*Conv) ToF(val interface{}) interface{} {
	switch v := val.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			panic(fmt.Sprintf("conv.to_f: cannot convert %q to float", v))
		}
		return f
	default:
		panic(fmt.Sprintf("conv.to_f: cannot convert %s to float", rugoTypeName(val)))
	}
}

func (*Conv) ToS(val interface{}) interface{} {
	return rugo_to_string(val)
}

func (*Conv) ToBool(val interface{}) interface{} {
	switch v := val.(type) {
	case bool:
		return v
	case int:
		return v != 0
	case float64:
		return v != 0.0
	case string:
		if v == "" || v == "false" {
			return false
		}
		return true
	case nil:
		return false
	default:
		panic(fmt.Sprintf("conv.to_bool: cannot convert %s to boolean", rugoTypeName(val)))
	}
}

func (*Conv) ParseInt(s string, base int) interface{} {
	n, err := strconv.ParseInt(s, base, 64)
	if err != nil {
		panic(fmt.Sprintf("conv.parse_int: cannot parse %q with base %d: %v", s, base, err))
	}
	return int(n)
}
