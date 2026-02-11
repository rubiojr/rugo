package gobridge

import "fmt"

var jsonHelpers = []RuntimeHelper{
	{Key: "rugo_json_prepare", Code: `func rugo_json_prepare(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val { m[fmt.Sprintf("%v", k)] = rugo_json_prepare(v) }
		return m
	case []interface{}:
		r := make([]interface{}, len(val))
		for i, e := range val { r[i] = rugo_json_prepare(e) }
		return r
	default:
		return v
	}
}

`},
	{Key: "rugo_json_to_rugo", Code: `func rugo_json_to_rugo(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[interface{}]interface{}, len(val))
		for k, v := range val { m[k] = rugo_json_to_rugo(v) }
		return m
	case []interface{}:
		r := make([]interface{}, len(val))
		for i, e := range val { r[i] = rugo_json_to_rugo(e) }
		return r
	default:
		return v
	}
}

`},
}

func init() {
	Register(&Package{
		Path: "encoding/json",
		Doc:  "JSON encoding and decoding functions from Go's encoding/json package.",
		Funcs: map[string]GoFuncSig{
			"marshal": {
				GoName: "Marshal", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Encodes a value to a JSON string.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.Marshal(rugo_json_prepare(%s)); if _err != nil { %s }; return interface{}(string(_v)) }()",
						pkgBase, args[0], PanicOnErr(rugoName))
				},
				RuntimeHelpers: jsonHelpers,
			},
			"unmarshal": {
				GoName: "Unmarshal", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Decodes a JSON string into a value.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { var _v interface{}; _err := %s.Unmarshal([]byte(rugo_to_string(%s)), &_v); if _err != nil { %s }; return rugo_json_to_rugo(_v) }()",
						pkgBase, args[0], PanicOnErr(rugoName))
				},
				RuntimeHelpers: jsonHelpers,
			},
			"marshal_indent": {
				GoName: "MarshalIndent", Params: []GoType{GoString, GoString, GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Encodes a value to a pretty-printed JSON string with prefix and indent.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf("func() interface{} { _v, _err := %s.MarshalIndent(rugo_json_prepare(%s), %s, %s); if _err != nil { %s }; return interface{}(string(_v)) }()",
						pkgBase, args[0], TypeConvToGo(args[1], GoString), TypeConvToGo(args[2], GoString), PanicOnErr(rugoName))
				},
				RuntimeHelpers: jsonHelpers,
			},
		},
	})
}
