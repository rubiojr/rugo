package cryptomod

import (
	_ "embed"

	"github.com/rubiojr/rugo/modules"
)

//go:embed runtime.go
var runtime string

func init() {
	modules.Register(&modules.Module{
		Name: "crypto",
		Type: "Crypto",
		Doc:  "Cryptographic hash functions.",
		Funcs: []modules.FuncDef{
			{Name: "md5", Args: []modules.ArgType{modules.String}, Doc: "Return the MD5 hex digest of a string."},
			{Name: "sha256", Args: []modules.ArgType{modules.String}, Doc: "Return the SHA-256 hex digest of a string."},
			{Name: "sha1", Args: []modules.ArgType{modules.String}, Doc: "Return the SHA-1 hex digest of a string."},
		},
		GoImports: []string{"crypto/md5", "crypto/sha1", "crypto/sha256"},
		Runtime:   modules.CleanRuntime(runtime),
	})
}
