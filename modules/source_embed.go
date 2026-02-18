package modules

import "embed"

// Sources embeds module.go and all module subdirectory source files needed
// to reconstruct the modules package tree in an external module cache.
// Test files are excluded.
//
//go:embed module.go
//go:embed ast/ast.go ast/runtime.go
//go:embed base64/base64.go base64/runtime.go
//go:embed bench/bench.go bench/runtime.go bench/stubs.go
//go:embed cli/cli.go cli/runtime.go cli/stubs.go
//go:embed color/color.go color/runtime.go
//go:embed conv/conv.go conv/runtime.go conv/stubs.go
//go:embed crypto/crypto.go crypto/runtime.go
//go:embed eval/eval.go eval/runtime.go eval/cache.go
//go:embed filepath/filepath.go filepath/runtime.go
//go:embed fmt/fmt.go fmt/runtime.go fmt/stubs.go
//go:embed hex/hex.go hex/runtime.go
//go:embed http/http.go http/runtime.go http/stubs.go
//go:embed json/json.go json/runtime.go
//go:embed math/math.go math/runtime.go
//go:embed os/os.go os/runtime.go
//go:embed queue/queue.go queue/runtime.go queue/stubs.go
//go:embed rand/rand.go rand/runtime.go
//go:embed re/re.go re/runtime.go re/stubs.go
//go:embed sqlite/runtime.go sqlite/sqlite.go
//go:embed str/runtime.go str/str.go
//go:embed test/runtime.go test/stubs.go test/test.go
//go:embed time/runtime.go time/time.go
//go:embed web/runtime.go web/stubs.go web/web.go
var Sources embed.FS
