package gobridge

func init() {
	Register(&Package{
		Path: "net/url",
		Doc:  "URL parsing and escaping functions from Go's net/url package.",
		Funcs: map[string]GoFuncSig{
			"parse": {
				GoName: "Parse", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Parses a URL string and returns a hash with scheme, host, hostname, port, path, query, fragment, user, and raw fields.",
				StructReturn: &GoStructReturn{
					Pointer: true,
					Fields: []GoStructField{
						{GoField: "Scheme", RugoKey: "scheme", Type: GoString},
						{GoField: "Host", RugoKey: "host", Type: GoString},
						{GoField: "Hostname", RugoKey: "hostname", Type: GoString, IsMethod: true},
						{GoField: "Port", RugoKey: "port", Type: GoString, IsMethod: true},
						{GoField: "Path", RugoKey: "path", Type: GoString},
						{GoField: "RawQuery", RugoKey: "query", Type: GoString},
						{GoField: "Fragment", RugoKey: "fragment", Type: GoString},
						{RugoKey: "user", Type: GoString, Expr: `func() string { if _v.User != nil { return _v.User.Username() }; return "" }()`},
						{GoField: "String", RugoKey: "raw", Type: GoString, IsMethod: true},
					},
				},
			},
			"path_escape":    {GoName: "PathEscape", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Escapes a string for use in a URL path segment."},
			"path_unescape":  {GoName: "PathUnescape", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}, Doc: "Unescapes a URL path segment."},
			"query_escape":   {GoName: "QueryEscape", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Escapes a string for use in a URL query parameter."},
			"query_unescape": {GoName: "QueryUnescape", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}, Doc: "Unescapes a URL query parameter."},
		},
	})
}
