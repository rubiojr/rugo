package gobridge

func init() {
	Register(&Package{
		Path: "time",
		Doc:  "Time functions from Go's time package.",
		Funcs: map[string]GoFuncSig{
			"now_unix": {
				GoName: "Now().Unix", Params: nil, Returns: []GoType{GoInt64},
				Doc: "Returns the current Unix timestamp in seconds.",
			},
			"now_unix_nano": {
				GoName: "Now().UnixNano", Params: nil, Returns: []GoType{GoInt64},
				Doc: "Returns the current Unix timestamp in nanoseconds.",
			},
			"sleep_ms": {GoName: "Sleep", Params: []GoType{GoDuration}, Returns: nil, Doc: "Sleeps for the given number of milliseconds."},
		},
	})
}
