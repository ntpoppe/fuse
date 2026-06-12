package config

import (
	"fmt"
	"net"

	"github.com/ntpoppe/fuse/internal/storage"
)

const (
	DefaultHost         = "127.0.0.1"
	DefaultPort         = 5000
	DefaultMaxQueryRows = 10_000
)

type Config struct {
	Host         string
	Port         int
	StateDBPath  string
	MaxQueryRows int
}

func NewConfig() *Config {
	return &Config{
		Host:         DefaultHost,
		StateDBPath:  storage.DefaultStateDBPath,
		MaxQueryRows: DefaultMaxQueryRows,
	}
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host must not be empty")
	}
	if ip := net.ParseIP(c.Host); ip == nil && c.Host != "localhost" {
		return fmt.Errorf("host %q is not a valid IP address or localhost", c.Host)
	}

	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", c.Port)
	}

	if c.Port < 1024 {
		fmt.Printf("warning: port %d is privileged; root access may be required\n", c.Port)
	}

	if c.StateDBPath == "" {
		return fmt.Errorf("state database path must not be empty")
	}

	if c.MaxQueryRows < 1 {
		return fmt.Errorf("max query rows must be at least 1")
	}

	return nil
}
