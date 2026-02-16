package hexmod

import (
	"encoding/hex"
)

// --- hex module ---

type Hex struct{}

func (*Hex) Encode(s string) interface{} {
	return hex.EncodeToString([]byte(s))
}

func (*Hex) Decode(s string) interface{} {
	data, err := hex.DecodeString(s)
	if err != nil {
		panic("hex.decode: " + err.Error())
	}
	return string(data)
}
