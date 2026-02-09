package gobridge

func init() {
	Register(&Package{
		Path: "math",
		Doc:  "Mathematical functions from Go's math package.",
		Funcs: map[string]GoFuncSig{
			"abs":       {GoName: "Abs", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the absolute value of x."},
			"ceil":      {GoName: "Ceil", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the smallest integer >= x."},
			"floor":     {GoName: "Floor", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the largest integer <= x."},
			"round":     {GoName: "Round", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the nearest integer, rounding half away from zero."},
			"max":       {GoName: "Max", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the larger of x or y."},
			"min":       {GoName: "Min", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the smaller of x or y."},
			"pow":       {GoName: "Pow", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns x raised to the power y."},
			"sqrt":      {GoName: "Sqrt", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the square root of x."},
			"cbrt":      {GoName: "Cbrt", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the cube root of x."},
			"log":       {GoName: "Log", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the natural logarithm of x."},
			"log2":      {GoName: "Log2", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the base-2 logarithm of x."},
			"log10":     {GoName: "Log10", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the base-10 logarithm of x."},
			"sin":       {GoName: "Sin", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the sine of x (radians)."},
			"cos":       {GoName: "Cos", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the cosine of x (radians)."},
			"tan":       {GoName: "Tan", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the tangent of x (radians)."},
			"asin":      {GoName: "Asin", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the arcsine of x in radians."},
			"acos":      {GoName: "Acos", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the arccosine of x in radians."},
			"atan":      {GoName: "Atan", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the arctangent of x in radians."},
			"atan2":     {GoName: "Atan2", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the arctangent of y/x in radians."},
			"exp":       {GoName: "Exp", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns e raised to the power x."},
			"mod":       {GoName: "Mod", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the floating-point remainder of x/y."},
			"remainder": {GoName: "Remainder", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}, Doc: "Returns the IEEE 754 remainder of x/y."},
			"inf":       {GoName: "Inf", Params: []GoType{GoInt}, Returns: []GoType{GoFloat64}, Doc: "Returns positive infinity if sign >= 0, negative infinity otherwise."},
			"is_inf":    {GoName: "IsInf", Params: []GoType{GoFloat64, GoInt}, Returns: []GoType{GoBool}, Doc: "Reports whether x is an infinity."},
			"is_nan":    {GoName: "IsNaN", Params: []GoType{GoFloat64}, Returns: []GoType{GoBool}, Doc: "Reports whether x is NaN."},
			"nan":       {GoName: "NaN", Params: nil, Returns: []GoType{GoFloat64}, Doc: "Returns an IEEE 754 NaN value."},
		},
	})
}
