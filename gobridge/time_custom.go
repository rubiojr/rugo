// Time bridge — all functions need custom annotations (method chains, GoDuration).
// Not auto-generatable by bridgegen.
package gobridge

func init() {
	Register(&Package{
		Path: "time",
		Doc:  "Time functions from Go's time package.",
		Funcs: map[string]GoFuncSig{
			"now_unix":       {GoName: "Now().Unix", Params: nil, Returns: []GoType{GoInt64}, Doc: "Returns the current Unix timestamp in seconds."},
			"now_unix_nano":  {GoName: "Now().UnixNano", Params: nil, Returns: []GoType{GoInt64}, Doc: "Returns the current Unix timestamp in nanoseconds."},
			"sleep_ms":       {GoName: "Sleep", Params: []GoType{GoDuration}, Returns: nil, Doc: "Sleeps for the given number of milliseconds."},
			"parse_duration": {GoName: "ParseDuration", Params: []GoType{GoString}, Returns: []GoType{GoDuration, GoError}, Doc: "Parses a duration string (e.g. \"300ms\", \"1.5h\", \"2h45m\") and returns milliseconds."},
		},
	})
}
