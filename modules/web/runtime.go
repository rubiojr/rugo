//go:build ignore

package webmod

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// --- web module ---

// routeEntry stores a registered route with its handler and middleware.
type routeEntry struct {
	method     string
	pattern    string
	segments   []string
	paramNames map[int]string
	handler    string
	middleware []string
	isStatic   bool
	staticDir  string
}

// Web is the Rugo web module providing chi-like HTTP routing.
type Web struct {
	routes           []routeEntry
	globalMiddleware []string
	groupPrefix      string
	groupMiddleware  []string
	inGroup          bool
	// rate limiter config
	rateLimitRPS float64
	rateLimiter  *tokenBucketLimiter
	// assigned port after listen
	listenPort  int
	listenReady chan struct{}
	readyOnce   sync.Once
}

// --- token bucket rate limiter ---

type tokenBucketLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	rps     float64
	burst   int
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
}

func newTokenBucketLimiter(rps float64) *tokenBucketLimiter {
	burst := int(rps)
	if burst < 1 {
		burst = 1
	}
	return &tokenBucketLimiter{
		buckets: make(map[string]*tokenBucket),
		rps:     rps,
		burst:   burst,
	}
}

func (l *tokenBucketLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	b, ok := l.buckets[key]
	if !ok {
		b = &tokenBucket{tokens: float64(l.burst), lastTime: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastTime).Seconds()
	b.tokens += elapsed * l.rps
	if b.tokens > float64(l.burst) {
		b.tokens = float64(l.burst)
	}
	b.lastTime = now

	if b.tokens >= 1.0 {
		b.tokens -= 1.0
		return true
	}
	return false
}

// --- Route registration ---

func (w *Web) Get(path string, handler string, extra ...interface{}) interface{} {
	w.addRoute("GET", path, handler, extra...)
	return nil
}

func (w *Web) Post(path string, handler string, extra ...interface{}) interface{} {
	w.addRoute("POST", path, handler, extra...)
	return nil
}

func (w *Web) Put(path string, handler string, extra ...interface{}) interface{} {
	w.addRoute("PUT", path, handler, extra...)
	return nil
}

func (w *Web) Delete(path string, handler string, extra ...interface{}) interface{} {
	w.addRoute("DELETE", path, handler, extra...)
	return nil
}

func (w *Web) Patch(path string, handler string, extra ...interface{}) interface{} {
	w.addRoute("PATCH", path, handler, extra...)
	return nil
}

// --- Middleware ---

func (w *Web) Middleware(name string) interface{} {
	w.globalMiddleware = append(w.globalMiddleware, name)
	return nil
}

// RateLimit configures the built-in rate limiter (requests per second per client IP).
// Must be called before web.middleware("rate_limiter").
func (w *Web) RateLimit(rps interface{}) interface{} {
	switch v := rps.(type) {
	case int:
		w.rateLimitRPS = float64(v)
	case float64:
		w.rateLimitRPS = v
	default:
		panic(fmt.Sprintf("web.rate_limit: expected number, got %T", rps))
	}
	if w.rateLimitRPS <= 0 {
		panic("web.rate_limit: requests per second must be > 0")
	}
	w.rateLimiter = newTokenBucketLimiter(w.rateLimitRPS)
	return nil
}

// --- Route groups ---

func (w *Web) Group(prefix string, extra ...interface{}) interface{} {
	w.groupPrefix = prefix
	w.groupMiddleware = nil
	w.inGroup = true
	for _, m := range extra {
		w.groupMiddleware = append(w.groupMiddleware, rugo_to_string(m))
	}
	return nil
}

func (w *Web) EndGroup() interface{} {
	w.groupPrefix = ""
	w.groupMiddleware = nil
	w.inGroup = false
	return nil
}

// --- Static file serving ---

func (w *Web) Static(urlPath string, dir string) interface{} {
	if !strings.HasSuffix(urlPath, "/") {
		urlPath += "/"
	}
	w.routes = append(w.routes, routeEntry{
		method:    "GET",
		pattern:   urlPath + "*",
		isStatic:  true,
		staticDir: dir,
	})
	return nil
}

// --- Response helpers ---

func (*Web) Text(body string, extra ...interface{}) interface{} {
	status := 200
	if len(extra) > 0 {
		status = rugo_to_int(extra[0])
	}
	return makeResponse(status, "text/plain; charset=utf-8", body)
}

func (*Web) Html(body string, extra ...interface{}) interface{} {
	status := 200
	if len(extra) > 0 {
		status = rugo_to_int(extra[0])
	}
	return makeResponse(status, "text/html; charset=utf-8", body)
}

func (w *Web) Json(data interface{}, extra ...interface{}) interface{} {
	status := 200
	if len(extra) > 0 {
		status = rugo_to_int(extra[0])
	}
	b, err := json.Marshal(prepareWebJSON(data))
	if err != nil {
		panic(fmt.Sprintf("web.json: %v", err))
	}
	return makeResponse(status, "application/json; charset=utf-8", string(b))
}

func (*Web) Redirect(url string, extra ...interface{}) interface{} {
	status := 302
	if len(extra) > 0 {
		status = rugo_to_int(extra[0])
	}
	resp := makeResponse(status, "", "")
	resp.(map[interface{}]interface{})["location"] = url
	return resp
}

func (*Web) Status(code int) interface{} {
	return makeResponse(code, "", "")
}

// --- Server ---

func (w *Web) Listen(port int) interface{} {
	handler := http.HandlerFunc(func(wr http.ResponseWriter, r *http.Request) {
		w.handleRequest(wr, r)
	})

	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		panic(fmt.Sprintf("web.listen: %v", err))
	}
	w.listenPort = ln.Addr().(*net.TCPAddr).Port
	w.readyOnce.Do(func() { w.listenReady = make(chan struct{}) })
	close(w.listenReady)
	if err := http.Serve(ln, handler); err != nil {
		panic(fmt.Sprintf("web.listen: %v", err))
	}
	return nil
}

// Port blocks until web.listen() has bound and returns the assigned port.
func (w *Web) Port() interface{} {
	w.readyOnce.Do(func() { w.listenReady = make(chan struct{}) })
	<-w.listenReady
	return w.listenPort
}

// FreePort asks the kernel for a free port and returns it.
func (*Web) FreePort() interface{} {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Sprintf("web.free_port: %v", err))
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// --- Internal: request handling ---

func (w *Web) handleRequest(wr http.ResponseWriter, r *http.Request) {
	// Match route
	route, params := w.matchRoute(r.Method, r.URL.Path)

	if route == nil {
		http.NotFound(wr, r)
		return
	}

	// Handle static files
	if route.isStatic {
		prefix := route.pattern[:len(route.pattern)-1] // strip trailing *
		relPath := strings.TrimPrefix(r.URL.Path, prefix)
		filePath := filepath.Join(route.staticDir, filepath.Clean("/"+relPath))
		http.ServeFile(wr, r, filePath)
		return
	}

	// Build request hash
	req := w.buildReqHash(r, params)

	// Run global middleware
	for _, mw := range w.globalMiddleware {
		result := w.callMiddleware(mw, req)
		if result != nil {
			w.writeResponse(wr, result)
			return
		}
	}

	// Run route-level middleware
	for _, mw := range route.middleware {
		result := w.callMiddleware(mw, req)
		if result != nil {
			w.writeResponse(wr, result)
			return
		}
	}

	// Call handler
	fn, ok := rugo_web_dispatch[route.handler]
	if !ok {
		http.Error(wr, fmt.Sprintf("web: no handler function %q defined", route.handler), 500)
		return
	}

	result := fn(req)
	if result == nil {
		wr.WriteHeader(204)
		return
	}
	w.writeResponse(wr, result)
}

func (w *Web) callMiddleware(name string, req interface{}) interface{} {
	// Built-in middleware
	switch name {
	case "logger":
		return w.mwLogger(req)
	case "recoverer":
		// recoverer is handled at the handler level — see handleRequest wrapper
		return nil
	case "real_ip":
		return w.mwRealIP(req)
	case "rate_limiter":
		return w.mwRateLimiter(req)
	}

	// User-defined middleware via dispatch
	fn, ok := rugo_web_dispatch[name]
	if !ok {
		fmt.Fprintf(os.Stderr, "web: warning: middleware %q not found\n", name)
		return nil
	}
	return fn(req)
}

// --- Built-in middleware ---

func (w *Web) mwLogger(req interface{}) interface{} {
	if m, ok := req.(map[interface{}]interface{}); ok {
		method := fmt.Sprintf("%v", m["method"])
		path := fmt.Sprintf("%v", m["path"])
		addr := fmt.Sprintf("%v", m["remote_addr"])
		log.Printf("%s %s %s", method, path, addr)
	}
	return nil
}

// mwRealIP extracts the real client IP from X-Forwarded-For or X-Real-Ip headers
// and overwrites req.remote_addr. Place before logger for accurate logging.
func (w *Web) mwRealIP(req interface{}) interface{} {
	m, ok := req.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	headers, _ := m["header"].(map[interface{}]interface{})
	if headers == nil {
		return nil
	}

	// Try X-Forwarded-For first (may contain comma-separated list)
	if xff, ok := headers["X-Forwarded-For"]; ok {
		if s := strings.TrimSpace(strings.SplitN(fmt.Sprintf("%v", xff), ",", 2)[0]); s != "" {
			m["remote_addr"] = s
			return nil
		}
	}

	// Fall back to X-Real-Ip
	if xri, ok := headers["X-Real-Ip"]; ok {
		if s := strings.TrimSpace(fmt.Sprintf("%v", xri)); s != "" {
			m["remote_addr"] = s
			return nil
		}
	}

	// Strip port from remote_addr as fallback normalization
	if addr, ok := m["remote_addr"].(string); ok {
		if host, _, err := net.SplitHostPort(addr); err == nil {
			m["remote_addr"] = host
		}
	}

	return nil
}

// mwRateLimiter enforces per-IP rate limiting using a token bucket algorithm.
// Configure with web.rate_limit(rps) before registering this middleware.
func (w *Web) mwRateLimiter(req interface{}) interface{} {
	if w.rateLimiter == nil {
		// Default: 10 requests/second if not configured
		w.rateLimiter = newTokenBucketLimiter(10)
	}

	m, ok := req.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	clientIP := fmt.Sprintf("%v", m["remote_addr"])
	// Strip port if present
	if host, _, err := net.SplitHostPort(clientIP); err == nil {
		clientIP = host
	}

	if !w.rateLimiter.allow(clientIP) {
		return makeResponse(429, "application/json; charset=utf-8", `{"error":"rate limit exceeded"}`)
	}

	return nil
}

// --- Internal: route matching ---

func (w *Web) addRoute(method, path, handler string, extra ...interface{}) {
	if w.inGroup {
		path = w.groupPrefix + path
	}

	segments := splitPath(path)
	paramNames := make(map[int]string)
	for i, seg := range segments {
		if strings.HasPrefix(seg, ":") {
			paramNames[i] = seg[1:]
		}
	}

	middleware := make([]string, 0)
	if w.inGroup {
		middleware = append(middleware, w.groupMiddleware...)
	}
	for _, m := range extra {
		middleware = append(middleware, rugo_to_string(m))
	}

	w.routes = append(w.routes, routeEntry{
		method:     method,
		pattern:    path,
		segments:   segments,
		paramNames: paramNames,
		handler:    handler,
		middleware: middleware,
	})
}

func (w *Web) matchRoute(method, path string) (*routeEntry, map[string]string) {
	reqSegments := splitPath(path)

	for i := range w.routes {
		route := &w.routes[i]
		if route.method != method {
			continue
		}

		// Static file routes
		if route.isStatic {
			prefix := route.pattern[:len(route.pattern)-1]
			if strings.HasPrefix(path, prefix) || path+"/" == prefix {
				return route, nil
			}
			continue
		}

		if len(route.segments) != len(reqSegments) {
			continue
		}

		params := make(map[string]string)
		matched := true
		for j, seg := range route.segments {
			if strings.HasPrefix(seg, ":") {
				params[seg[1:]] = reqSegments[j]
			} else if seg != reqSegments[j] {
				matched = false
				break
			}
		}
		if matched {
			return route, params
		}
	}
	return nil, nil
}

// --- Internal: request building ---

func (w *Web) buildReqHash(r *http.Request, params map[string]string) interface{} {
	// Read body
	var body string
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err == nil {
			body = string(b)
		}
		r.Body.Close()
	}

	// Build params hash
	paramsHash := make(map[interface{}]interface{})
	for k, v := range params {
		paramsHash[k] = v
	}

	// Build query hash
	queryHash := make(map[interface{}]interface{})
	for k, v := range r.URL.Query() {
		if len(v) == 1 {
			queryHash[k] = v[0]
		} else {
			arr := make([]interface{}, len(v))
			for i, s := range v {
				arr[i] = s
			}
			queryHash[k] = arr
		}
	}

	// Build header hash
	headerHash := make(map[interface{}]interface{})
	for k, v := range r.Header {
		if len(v) == 1 {
			headerHash[k] = v[0]
		} else {
			headerHash[k] = strings.Join(v, ", ")
		}
	}

	return map[interface{}]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"body":        body,
		"params":      paramsHash,
		"query":       queryHash,
		"header":      headerHash,
		"remote_addr": r.RemoteAddr,
	}
}

// --- Internal: response writing ---

func (w *Web) writeResponse(wr http.ResponseWriter, result interface{}) {
	resp, ok := result.(map[interface{}]interface{})
	if !ok {
		// Plain string response
		wr.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(wr, rugo_to_string(result))
		return
	}

	status := 200
	if s, ok := resp["status"]; ok {
		status = rugo_to_int(s)
	}

	if ct, ok := resp["content_type"]; ok {
		if s := rugo_to_string(ct); s != "" {
			wr.Header().Set("Content-Type", s)
		}
	}

	// Custom headers
	if hdrs, ok := resp["headers"]; ok {
		if hm, ok := hdrs.(map[interface{}]interface{}); ok {
			for k, v := range hm {
				wr.Header().Set(rugo_to_string(k), rugo_to_string(v))
			}
		}
	}

	// Redirect
	if loc, ok := resp["location"]; ok {
		http.Redirect(wr, &http.Request{}, rugo_to_string(loc), status)
		return
	}

	wr.WriteHeader(status)

	if body, ok := resp["body"]; ok {
		fmt.Fprint(wr, rugo_to_string(body))
	}
}

// --- Internal helpers ---

func splitPath(path string) []string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 1 && parts[0] == "" {
		return []string{}
	}
	return parts
}

func makeResponse(status int, contentType, body string) interface{} {
	return map[interface{}]interface{}{
		"__type__":     "Response",
		"status":       status,
		"content_type": contentType,
		"body":         body,
	}
}

// prepareWebJSON converts Rugo maps to Go maps for JSON marshaling.
func prepareWebJSON(v interface{}) interface{} {
	switch val := v.(type) {
	case map[interface{}]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, child := range val {
			m[fmt.Sprintf("%v", k)] = prepareWebJSON(child)
		}
		return m
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, child := range val {
			out[i] = prepareWebJSON(child)
		}
		return out
	default:
		return v
	}
}

// Silence unused import warnings — these are used by the generated program.
var _ = time.Now
var _ = math.MaxInt
var _ = log.Printf
var _ sync.Mutex
var _ = net.SplitHostPort
