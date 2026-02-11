package parser

import _ "embed"

// Source contains the parser source code for embedding into the AST cache.
//
//go:embed parser.go
var Source []byte
