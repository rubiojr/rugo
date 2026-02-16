package cryptomod

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"fmt"
)

// --- crypto module ---

type Crypto struct{}

func (*Crypto) Md5(s string) interface{} {
	h := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}

func (*Crypto) Sha256(s string) interface{} {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func (*Crypto) Sha1(s string) interface{} {
	h := sha1.Sum([]byte(s))
	return fmt.Sprintf("%x", h)
}
