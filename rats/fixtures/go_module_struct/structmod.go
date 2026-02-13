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
