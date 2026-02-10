package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "net/url",
		Doc:  "URL parsing and escaping functions from Go's net/url package.",
		Funcs: map[string]GoFuncSig{
			"parse": {
				GoName: "Parse", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError},
				Doc: "Parses a URL string and returns a hash with scheme, host, hostname, port, path, query, fragment, user, and raw fields.",
				Codegen: func(pkgBase string, args []string, rugoName string) string {
					return fmt.Sprintf(`func() interface{} {
	_u, _err := %s.Parse(%s)
	if _err != nil { %s }
	_user := ""
	if _u.User != nil { _user = _u.User.Username() }
	return map[interface{}]interface{}{
		"scheme": interface{}(_u.Scheme),
		"host": interface{}(_u.Host),
		"hostname": interface{}(_u.Hostname()),
		"port": interface{}(_u.Port()),
		"path": interface{}(_u.Path),
		"query": interface{}(_u.RawQuery),
		"fragment": interface{}(_u.Fragment),
		"user": interface{}(_user),
		"raw": interface{}(_u.String()),
	}
}()`, pkgBase, TypeConvToGo(args[0], GoString), PanicOnErr(rugoName))
				},
			},
			"path_escape":    {GoName: "PathEscape", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Escapes a string for use in a URL path segment."},
			"path_unescape":  {GoName: "PathUnescape", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}, Doc: "Unescapes a URL path segment."},
			"query_escape":   {GoName: "QueryEscape", Params: []GoType{GoString}, Returns: []GoType{GoString}, Doc: "Escapes a string for use in a URL query parameter."},
			"query_unescape": {GoName: "QueryUnescape", Params: []GoType{GoString}, Returns: []GoType{GoString, GoError}, Doc: "Unescapes a URL query parameter."},
		},
	})
}
