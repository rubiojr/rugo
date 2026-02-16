package base64mod

import (
	"encoding/base64"
)

// --- base64 module ---

type Base64 struct{}

func (*Base64) Encode(s string) interface{} {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

func (*Base64) Decode(s string) interface{} {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		panic("base64.decode: " + err.Error())
	}
	return string(data)
}

func (*Base64) UrlEncode(s string) interface{} {
	return base64.URLEncoding.EncodeToString([]byte(s))
}

func (*Base64) UrlDecode(s string) interface{} {
	data, err := base64.URLEncoding.DecodeString(s)
	if err != nil {
		panic("base64.url_decode: " + err.Error())
	}
	return string(data)
}
