package slug

import "fmt"

var _ = rugo_to_string

func rugo_to_string(v interface{}) string { return fmt.Sprintf("%v", v) }
