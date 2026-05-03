package config

import (
	"fmt"
)

type Config struct {
	Port int
	Env  string
}

func NewConfig() *Config {
	config := Config{}
	return &config
}

func (c *Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d is out of range (1-65535)", c.Port)
	}

	if c.Port < 1024 {
		fmt.Printf("Warning: Port %d is a privileged port; root access may be required\n", c.Port)
	}

	allowedEnvs := map[string]bool{"dev": true, "prod": true}
	if !allowedEnvs[c.Env] {
		return fmt.Errorf("invalid environment %q: must be 'dev', 'prod'", c.Env)
	}

	return nil
}
