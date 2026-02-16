package mathmod

import (
	"math"
	"math/rand/v2"
)

// --- math module ---

type Math struct{}

func (*Math) Abs(n float64) interface{} {
	return math.Abs(n)
}

func (*Math) Ceil(n float64) interface{} {
	return int(math.Ceil(n))
}

func (*Math) Floor(n float64) interface{} {
	return int(math.Floor(n))
}

func (*Math) Round(n float64) interface{} {
	return int(math.Round(n))
}

func (*Math) Max(a, b float64) interface{} {
	return math.Max(a, b)
}

func (*Math) Min(a, b float64) interface{} {
	return math.Min(a, b)
}

func (*Math) Pow(base, exp float64) interface{} {
	return math.Pow(base, exp)
}

func (*Math) Sqrt(n float64) interface{} {
	return math.Sqrt(n)
}

func (*Math) Log(n float64) interface{} {
	return math.Log(n)
}

func (*Math) Log2(n float64) interface{} {
	return math.Log2(n)
}

func (*Math) Log10(n float64) interface{} {
	return math.Log10(n)
}

func (*Math) Sin(n float64) interface{} {
	return math.Sin(n)
}

func (*Math) Cos(n float64) interface{} {
	return math.Cos(n)
}

func (*Math) Tan(n float64) interface{} {
	return math.Tan(n)
}

func (*Math) Pi() interface{} {
	return math.Pi
}

func (*Math) E() interface{} {
	return math.E
}

func (*Math) Inf() interface{} {
	return math.Inf(1)
}

func (*Math) Nan() interface{} {
	return math.NaN()
}

func (*Math) IsNan(n float64) interface{} {
	return math.IsNaN(n)
}

func (*Math) IsInf(n float64) interface{} {
	return math.IsInf(n, 0)
}

func (*Math) Clamp(n, min, max float64) interface{} {
	return math.Max(min, math.Min(max, n))
}

func (*Math) Random() interface{} {
	return rand.Float64()
}

func (*Math) RandomInt(min, max int) interface{} {
	return rand.IntN(max-min) + min
}
