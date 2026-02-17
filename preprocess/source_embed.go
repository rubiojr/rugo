package preprocess

import "embed"

//go:embed preprocess.go string_tracker.go
var Sources embed.FS
