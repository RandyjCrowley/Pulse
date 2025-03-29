package config

import (
	"flag"
)

// Config holds application configuration
type Config struct {
	Debug bool
}

// ParseFlags parses command line flags and returns config
func ParseFlags() Config {
	debug := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	return Config{
		Debug: *debug,
	}
}
