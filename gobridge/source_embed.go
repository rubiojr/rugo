package gobridge

import "embed"

// Sources embeds all Go source files needed to reconstruct
// the gobridge package in an external module cache.
//
//go:embed *.go helpers/*.go
var Sources embed.FS
