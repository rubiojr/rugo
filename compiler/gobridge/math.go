package gobridge

func init() {
	Register(&Package{
		Path: "math",
		Funcs: map[string]GoFuncSig{
			"abs":       {GoName: "Abs", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"ceil":      {GoName: "Ceil", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"floor":     {GoName: "Floor", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"round":     {GoName: "Round", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"max":       {GoName: "Max", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"min":       {GoName: "Min", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"pow":       {GoName: "Pow", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"sqrt":      {GoName: "Sqrt", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"cbrt":      {GoName: "Cbrt", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"log":       {GoName: "Log", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"log2":      {GoName: "Log2", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"log10":     {GoName: "Log10", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"sin":       {GoName: "Sin", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"cos":       {GoName: "Cos", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"tan":       {GoName: "Tan", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"asin":      {GoName: "Asin", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"acos":      {GoName: "Acos", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"atan":      {GoName: "Atan", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"atan2":     {GoName: "Atan2", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"exp":       {GoName: "Exp", Params: []GoType{GoFloat64}, Returns: []GoType{GoFloat64}},
			"mod":       {GoName: "Mod", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"remainder": {GoName: "Remainder", Params: []GoType{GoFloat64, GoFloat64}, Returns: []GoType{GoFloat64}},
			"inf":       {GoName: "Inf", Params: []GoType{GoInt}, Returns: []GoType{GoFloat64}},
			"is_inf":    {GoName: "IsInf", Params: []GoType{GoFloat64, GoInt}, Returns: []GoType{GoBool}},
			"is_nan":    {GoName: "IsNaN", Params: []GoType{GoFloat64}, Returns: []GoType{GoBool}},
			"nan":       {GoName: "NaN", Params: nil, Returns: []GoType{GoFloat64}},
		},
	})
}
