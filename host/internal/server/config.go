package server

// Config holds server configuration.
type Config struct {
	Bind   string
	Port   string
	Token  string
	LogDir string
	WebDir string
}
