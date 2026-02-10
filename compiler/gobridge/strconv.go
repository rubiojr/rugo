package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "strconv",
		Doc:  "String conversion functions from Go's strconv package.",
		Funcs: map[string]GoFuncSig{
			"atoi":         {GoName: "Atoi", Params: []GoType{GoString}, Returns: []GoType{GoInt, GoError}, Doc: "Converts a string to an integer."},
			"itoa":         {GoName: "Itoa", Params: []GoType{GoInt}, Returns: []GoType{GoString}, Doc: "Converts an integer to a string."},
			"format_bool":  {GoName: "FormatBool", Params: []GoType{GoBool}, Returns: []GoType{GoString}, Doc: "Returns \"true\" or \"false\" for the boolean value."},
			"format_float": {GoName: "FormatFloat", Params: []GoType{GoFloat64, GoByte, GoInt, GoInt}, Returns: []GoType{GoString}, Doc: "Converts a float to a string with the given format and precision."},
			"parse_bool":   {GoName: "ParseBool", Params: []GoType{GoString}, Returns: []GoType{GoBool, GoError}, Doc: "Returns the boolean value represented by the string."},
			"parse_float":  {GoName: "ParseFloat", Params: []GoType{GoString, GoInt}, Returns: []GoType{GoFloat64, GoError}, Doc: "Converts a string to a floating-point number."},
			"format_int": {
				GoName: "FormatInt", Params: []GoType{GoInt, GoInt}, Returns: []GoType{GoString},
				Doc: "Returns the string representation of i in the given base.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("interface{}(%s.FormatInt(int64(%s), %s))",
						pkgBase, TypeConvToGo(args[0], GoInt), TypeConvToGo(args[1], GoInt))
				},
			},
			"parse_int": {
				GoName: "ParseInt", Params: []GoType{GoString, GoInt, GoInt}, Returns: []GoType{GoInt, GoError},
				Doc: "Interprets a string in the given base and bit size.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.ParseInt(%s, %s, %s); if _err != nil { %s }; return interface{}(int(_v)) }()",
						pkgBase, TypeConvToGo(args[0], GoString), TypeConvToGo(args[1], GoInt), TypeConvToGo(args[2], GoInt), PanicOnErr(rugoName))
				},
			},
		},
	})
}
