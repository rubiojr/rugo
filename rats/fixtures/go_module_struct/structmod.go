package structmod

// Config is a simple struct with string and int fields.
type Config struct {
	Name string
	Port int
}

// Server has multiple field types.
type Server struct {
	Host    string
	Port    int
	Debug   bool
	Timeout float64
}

// Counter is an opaque struct with no exported fields.
type Counter struct {
	count int
}

// NewConfig creates a Config with the given name and port.
func NewConfig(name string, port int) *Config {
	return &Config{Name: name, Port: port}
}

// GetName returns the name from a Config.
func GetName(c *Config) string {
	return c.Name
}

// GetPort returns the port from a Config.
func GetPort(c *Config) int {
	return c.Port
}

// SetName sets the name on a Config and returns the updated Config.
func SetName(c *Config, name string) *Config {
	c.Name = name
	return c
}

// Describe returns a summary string from a Config.
func Describe(c *Config) string {
	return c.Name + " config"
}

// NewServer creates a Server with the given host and port.
func NewServer(host string, port int) *Server {
	return &Server{Host: host, Port: port}
}

// ServerHost returns the host from a Server.
func ServerHost(s *Server) string {
	return s.Host
}

// IsDebug returns the debug flag from a Server.
func IsDebug(s *Server) bool {
	return s.Debug
}

// Greet is a plain function (no structs) to verify mixed packages work.
func Greet(name string) string {
	return "Hello, " + name + "!"
}

// --- Methods on Config ---

// Summary returns a description of the config.
func (c *Config) Summary() string {
	return c.Name + " config"
}

// SetPort updates the port.
func (c *Config) SetPort(port int) {
	c.Port = port
}

// Clone returns a copy of the config.
func (c *Config) Clone() *Config {
	return &Config{Name: c.Name, Port: c.Port}
}

// --- Methods on Server ---

// Address returns host:port as a string.
func (s *Server) Address() string {
	return s.Host + ":" + Itoa(s.Port)
}

// SetDebug sets the debug flag.
func (s *Server) SetDebug(debug bool) {
	s.Debug = debug
}

// --- Methods on Counter (opaque) ---

// NewCounter creates a counter starting at zero.
func NewCounter() *Counter {
	return &Counter{count: 0}
}

// Inc increments the counter.
func (c *Counter) Inc() {
	c.count++
}

// Add adds n to the counter.
func (c *Counter) Add(n int) {
	c.count += n
}

// Value returns the current count.
func (c *Counter) Value() int {
	return c.count
}

// Reset resets to zero.
func (c *Counter) Reset() {
	c.count = 0
}

// Itoa is a helper (not using strconv to keep go.mod minimal).
func Itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
