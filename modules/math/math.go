package mathmod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "math",
		Type: "Math",
		Doc:  "Mathematical functions and constants.",
		Funcs: []modules.FuncDef{
			{Name: "abs", Args: []modules.ArgType{modules.Float}, Doc: "Return the absolute value of n."},
			{Name: "ceil", Args: []modules.ArgType{modules.Float}, Doc: "Round n up to the nearest integer."},
			{Name: "floor", Args: []modules.ArgType{modules.Float}, Doc: "Round n down to the nearest integer."},
			{Name: "round", Args: []modules.ArgType{modules.Float}, Doc: "Round n to the nearest integer."},
			{Name: "max", Args: []modules.ArgType{modules.Float, modules.Float}, Doc: "Return the larger of a and b."},
			{Name: "min", Args: []modules.ArgType{modules.Float, modules.Float}, Doc: "Return the smaller of a and b."},
			{Name: "pow", Args: []modules.ArgType{modules.Float, modules.Float}, Doc: "Return base raised to the power of exp."},
			{Name: "sqrt", Args: []modules.ArgType{modules.Float}, Doc: "Return the square root of n."},
			{Name: "log", Args: []modules.ArgType{modules.Float}, Doc: "Return the natural logarithm of n."},
			{Name: "log2", Args: []modules.ArgType{modules.Float}, Doc: "Return the base-2 logarithm of n."},
			{Name: "log10", Args: []modules.ArgType{modules.Float}, Doc: "Return the base-10 logarithm of n."},
			{Name: "sin", Args: []modules.ArgType{modules.Float}, Doc: "Return the sine of n (radians)."},
			{Name: "cos", Args: []modules.ArgType{modules.Float}, Doc: "Return the cosine of n (radians)."},
			{Name: "tan", Args: []modules.ArgType{modules.Float}, Doc: "Return the tangent of n (radians)."},
			{Name: "pi", Doc: "Return the value of Pi."},
			{Name: "e", Doc: "Return the value of Euler's number (e)."},
			{Name: "inf", Doc: "Return positive infinity."},
			{Name: "nan", Doc: "Return NaN (not a number)."},
			{Name: "is_nan", Args: []modules.ArgType{modules.Float}, Doc: "Return true if n is NaN."},
			{Name: "is_inf", Args: []modules.ArgType{modules.Float}, Doc: "Return true if n is infinite."},
			{Name: "clamp", Args: []modules.ArgType{modules.Float, modules.Float, modules.Float}, Doc: "Clamp n between min and max."},
			{Name: "random", Doc: "Return a random float in [0.0, 1.0)."},
			{Name: "random_int", Args: []modules.ArgType{modules.Int, modules.Int}, Doc: "Return a random integer in [min, max)."},
		},
		GoImports: []string{"math", "math/rand/v2"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
