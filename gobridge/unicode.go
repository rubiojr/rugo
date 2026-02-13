package gobridge

func init() {
	Register(&Package{
		Path: "unicode",
		Doc:  "Character classification and case conversion functions from Go's unicode package.",
		Funcs: map[string]GoFuncSig{
			"is_letter": {GoName: "IsLetter", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is a letter."},
			"is_digit":  {GoName: "IsDigit", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is a digit."},
			"is_space":  {GoName: "IsSpace", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is whitespace."},
			"is_upper":  {GoName: "IsUpper", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is uppercase."},
			"is_lower":  {GoName: "IsLower", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is lowercase."},
			"is_punct":  {GoName: "IsPunct", Params: []GoType{GoRune}, Returns: []GoType{GoBool}, Doc: "Reports whether the first character is punctuation."},
			"to_upper":  {GoName: "ToUpper", Params: []GoType{GoRune}, Returns: []GoType{GoRune}, Doc: "Converts the first character to uppercase."},
			"to_lower":  {GoName: "ToLower", Params: []GoType{GoRune}, Returns: []GoType{GoRune}, Doc: "Converts the first character to lowercase."},
		},
	})
}
