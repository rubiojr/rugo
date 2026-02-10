package gobridge

import "fmt"

func init() {
	Register(&Package{
		Path: "time",
		Doc:  "Time functions from Go's time package.",
		Funcs: map[string]GoFuncSig{
			"now_unix": {
				GoName: "Now().Unix", Params: nil, Returns: []GoType{GoInt},
				Doc: "Returns the current Unix timestamp in seconds.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("interface{}(int(%s.Now().Unix()))", pkgBase)
				},
			},
			"now_unix_nano": {
				GoName: "Now().UnixNano", Params: nil, Returns: []GoType{GoInt},
				Doc: "Returns the current Unix timestamp in nanoseconds.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("interface{}(int(%s.Now().UnixNano()))", pkgBase)
				},
			},
			"sleep_ms": {
				GoName: "Sleep", Params: []GoType{GoInt}, Returns: nil,
				Doc: "Sleeps for the given number of milliseconds.",
				Codegen: func(pkgBase string, args []string, _ string) string {
					return fmt.Sprintf("func() interface{} { %s.Sleep(time.Duration(%s) * time.Millisecond); return nil }()",
						pkgBase, TypeConvToGo(args[0], GoInt))
				},
			},
		},
	})
}
