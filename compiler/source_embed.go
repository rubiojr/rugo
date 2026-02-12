package compiler

import "embed"

// Sources embeds all non-test Go source files and templates needed to
// reconstruct the compiler package in an external module cache.
//
//go:embed compiler.go codegen.go ext.go infer.go types.go visitor.go
//go:embed templates/runtime_core_pre.go.tmpl templates/runtime_core_post.go.tmpl templates/runtime_spawn.go.tmpl
var Sources embed.FS
