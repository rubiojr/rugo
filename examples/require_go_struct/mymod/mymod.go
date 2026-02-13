package mymod

// Config holds application configuration.
type Config struct {
	Name string
	Port int
}

// NewConfig creates a Config with the given name and port.
func NewConfig(name string, port int) *Config {
	return &Config{Name: name, Port: port}
}

// Describe returns a human-readable description of the config.
func Describe(c *Config) string {
	return c.Name + " server"
}
