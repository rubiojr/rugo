package gobridge

import "fmt"

var utf8Helper = []RuntimeHelper{
	{Key: "rugo_utf8_decode", Code: `func rugo_utf8_decode(s string) (rune, int) {
	for _, r := range s { return r, 0 }
	return 0, 0
}

`},
}

func unicodePredicate(goName string) CodegenFunc {
	return func(pkgBase string, args []string, rugoName string) string {
		return fmt.Sprintf(`func() interface{} { _s := %s; if len(_s) == 0 { return interface{}(false) }; _r, _ := rugo_utf8_decode(_s); return interface{}(%s.%s(_r)) }()`,
			TypeConvToGo(args[0], GoString), pkgBase, goName)
	}
}

func unicodeCaseConvert(goName string) CodegenFunc {
	return func(pkgBase string, args []string, rugoName string) string {
		return fmt.Sprintf(`func() interface{} { _s := %s; if len(_s) == 0 { return interface{}("") }; _r, _ := rugo_utf8_decode(_s); return interface{}(string(%s.%s(_r))) }()`,
			TypeConvToGo(args[0], GoString), pkgBase, goName)
	}
}

func init() {
	Register(&Package{
		Path: "unicode",
		Doc:  "Character classification and case conversion functions from Go's unicode package.",
		Funcs: map[string]GoFuncSig{
			"is_letter": {GoName: "IsLetter", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is a letter.", Codegen: unicodePredicate("IsLetter"), RuntimeHelpers: utf8Helper},
			"is_digit":  {GoName: "IsDigit", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is a digit.", Codegen: unicodePredicate("IsDigit"), RuntimeHelpers: utf8Helper},
			"is_space":  {GoName: "IsSpace", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is whitespace.", Codegen: unicodePredicate("IsSpace"), RuntimeHelpers: utf8Helper},
			"is_upper":  {GoName: "IsUpper", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is uppercase.", Codegen: unicodePredicate("IsUpper"), RuntimeHelpers: utf8Helper},
			"is_lower":  {GoName: "IsLower", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is lowercase.", Codegen: unicodePredicate("IsLower"), RuntimeHelpers: utf8Helper},
			"is_punct":  {GoName: "IsPunct", Params: []GoType{GoString}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is punctuation.", Codegen: unicodePredicate("IsPunct"), RuntimeHelpers: utf8Helper},
			"to_upper":  {GoName: "ToUpper", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Converts the first character to uppercase.", Codegen: unicodeCaseConvert("ToUpper"), RuntimeHelpers: utf8Helper},
			"to_lower":  {GoName: "ToLower", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Converts the first character to lowercase.", Codegen: unicodeCaseConvert("ToLower"), RuntimeHelpers: utf8Helper},
		},
	})
}
