package remote

import "embed"

// Sources embeds all non-test Go source files needed to reconstruct
// the remote package in an external module cache.
//
//go:embed lockfile.go remote.go resolver.go
var Sources embed.FS
