package config

import (
	"fmt"

	"github.com/ntpoppe/fuse/internal/storage"
)

const (
	DefaultPort         = 5000
	DefaultMaxQueryRows = 10_000
)

type Config struct {
	Port         int
	StateDBPath  string
	MaxQueryRows int
}

func NewConfig() *Config {
	return &Config{
		StateDBPath:  storage.DefaultStateDBPath,
		MaxQueryRows: DefaultMaxQueryRows,
	}
}

func (c *Config) Validate() error {
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
