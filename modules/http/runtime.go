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

// httpErr simplifies a Go net/http error for human-friendly output.
func httpErr(funcName string, err error) string {
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

// doRequest builds and executes an HTTP request, returning a Rugo response hash.
func doRequest(method, rawURL string, body string, headers map[interface{}]interface{}) map[interface{}]interface{} {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, rawURL, bodyReader)
	if err != nil {
		panic(httpErr("http."+strings.ToLower(method), err))
	}

	// Set default Content-Type for methods that carry a body
	if body != "" && (method == "POST" || method == "PUT" || method == "PATCH") {
		req.Header.Set("Content-Type", "application/json")
	}

	for k, v := range headers {
		req.Header.Set(rugo_to_string(k), rugo_to_string(v))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(httpErr("http."+strings.ToLower(method), err))
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(fmt.Sprintf("http.%s: failed to read response body: %v", strings.ToLower(method), err))
	}

	respHeaders := make(map[interface{}]interface{})
	for k, vals := range resp.Header {
		respHeaders[k] = strings.Join(vals, ", ")
	}

	return map[interface{}]interface{}{
		"status_code": resp.StatusCode,
		"body":        string(respBody),
		"headers":     respHeaders,
	}
}

// extractHeaders extracts an optional headers hash from variadic extra args.
func extractHeaders(extra []interface{}) map[interface{}]interface{} {
	for _, arg := range extra {
		if h, ok := arg.(map[interface{}]interface{}); ok {
			return h
		}
	}
	return nil
}

func (*HTTP) Get(url string, extra ...interface{}) interface{} {
	headers := extractHeaders(extra)
	return doRequest("GET", url, "", headers)
}

func (*HTTP) Post(url string, body string, extra ...interface{}) interface{} {
	headers := extractHeaders(extra)
	return doRequest("POST", url, body, headers)
}

func (*HTTP) Put(url string, body string, extra ...interface{}) interface{} {
	headers := extractHeaders(extra)
	return doRequest("PUT", url, body, headers)
}

func (*HTTP) Patch(url string, body string, extra ...interface{}) interface{} {
	headers := extractHeaders(extra)
	return doRequest("PATCH", url, body, headers)
}

func (*HTTP) Delete(url string, extra ...interface{}) interface{} {
	headers := extractHeaders(extra)
	return doRequest("DELETE", url, "", headers)
}
