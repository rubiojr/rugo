//go:build ignore

package convmod

// --- conv module ---

type Conv struct{}

func (*Conv) ToI(val interface{}) interface{} {
	switch v := val.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		n, err := strconv.Atoi(v)
		if err != nil {
			panic(fmt.Sprintf("conv.to_i: cannot convert %q to int", v))
		}
		return n
	case bool:
		if v {
			return 1
		}
		return 0
	default:
		panic(fmt.Sprintf("conv.to_i: cannot convert %T to int", val))
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
		panic(fmt.Sprintf("conv.to_f: cannot convert %T to float", val))
	}
}

func (*Conv) ToS(val interface{}) interface{} {
	return rugo_to_string(val)
}
