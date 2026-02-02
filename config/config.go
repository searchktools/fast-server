package config

import (
	"flag"
	"os"
)

// Config holds all application configuration.
type Config struct {
	Port         int
	ReadTimeout  int
	WriteTimeout int
	Env          string
}

// New loads configuration from flags (and potentially env vars).
func New() *Config {
	cfg := &Config{}

	flag.IntVar(&cfg.Port, "port", 8080, "HTTP server port")
	flag.IntVar(&cfg.ReadTimeout, "read-timeout", 10, "HTTP read timeout (seconds)")
	flag.IntVar(&cfg.WriteTimeout, "write-timeout", 30, "HTTP write timeout (seconds)")
	flag.StringVar(&cfg.Env, "env", "development", "Environment (development/production)")

	flag.Parse()

	// Example: Override with ENV if present
	if port := os.Getenv("PORT"); port != "" {
		// logic to parse port string to int...
	}

	return cfg
}
