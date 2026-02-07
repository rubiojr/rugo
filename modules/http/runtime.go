package httpmod

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// --- http module ---

type HTTP struct{}

var _ = io.Discard
var _ = http.Get

func (*HTTP) Get(url string) interface{} {
	resp, err := http.Get(url)
	if err != nil {
		panic(fmt.Sprintf("http.get failed: %v", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.get read failed: %v", err))
	}
	return string(body)
}

func (*HTTP) Post(url string, body string, extra ...interface{}) interface{} {
	contentType := "application/json"
	if len(extra) > 0 {
		contentType = rugo_to_string(extra[0])
	}
	resp, err := http.Post(url, contentType, strings.NewReader(body))
	if err != nil {
		panic(fmt.Sprintf("http.post failed: %v", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.post read failed: %v", err))
	}
	return string(respBody)
}
