package gobridge

import "embed"

// Sources embeds all non-test Go source files needed to reconstruct
// the gobridge package in an external module cache.
//
//go:embed base64.go crypto.go filepath.go gobridge.go hex.go json.go maps.go math.go os.go rand.go slices.go sort.go strconv.go strings.go time.go unicode.go url.go
var Sources embed.FS
