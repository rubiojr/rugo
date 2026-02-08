package httpmod

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// --- http module ---

type HTTP struct{}

var _ = io.Discard
var _ = http.Get

// httpErr simplifies a Go net/http error for human-friendly output.
func httpErr(funcName string, err error) string {
	// Unwrap URL errors to get the inner error
	var urlErr *url.Error
	if ok := errors.As(err, &urlErr); ok {
		err = urlErr.Err
	}
	var netErr *net.OpError
	if ok := errors.As(err, &netErr); ok {
		err = netErr.Err
	}
	return fmt.Sprintf("%s failed: %v", funcName, err)
}

func (*HTTP) Get(url string) interface{} {
	resp, err := http.Get(url)
	if err != nil {
		panic(httpErr("http.get", err))
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.get: failed to read response body: %v", err))
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
		panic(httpErr("http.post", err))
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.post: failed to read response body: %v", err))
	}
	return string(respBody)
}
